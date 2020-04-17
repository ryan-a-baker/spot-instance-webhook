package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	v1 "k8s.io/kubernetes/pkg/apis/core/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

var (
	ignoredNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}
)

var tolerationToAdd = corev1.Toleration{
	Key:      "spot",
	Value:    "true",
	Operator: "Equal",
	Effect:   "NoSchedule",
}

type WebhookServer struct {
	server *http.Server
}

// Webhook Server parameters
type WhSvrParameters struct {
	port           int    // webhook server port
	certFile       string // path to the x509 certificate for https
	keyFile        string // path to the x509 private key matching `CertFile`
	sidecarCfgFile string // path to sidecar injector configuration file
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)
}

func isNameSpaceIgnored(ignoredList []string, reqNamespace string) bool {
	// Skip system or requested namespaces
	for _, namespace := range ignoredList {
		if reqNamespace == namespace {
			return true
		}
	}
	return false
}

// Check to see if the toleration for spot instances already exists on the resource,
// if so, return false as we ddon't need to make any adjustments
func tolerationAlreadyExists(existingTolerations []corev1.Toleration) bool {
	for _, tol := range existingTolerations {
		if tol.MatchToleration(&tolerationToAdd) {
			glog.Infof("Toleration already exists on the resource")
			return true
		}
	}
	return false
}

// Check to see if the node selector for the spot instance nodes already exists on the
// resource, if so, return false, else return true
func selectorAlreadyExists(existingNodeSelector map[string]string) bool {
	if existingNodeSelector["spot"] == "true" {
		glog.Infof("Node Selector already exists on the resource")
		return true
	}
	return false
}

// This may just be able to be Tolerations
func updateTolerations(existingTolerations []corev1.Toleration) (patch []patchOperation) {

	var updateNeeded = true

	if updateNeeded {
		glog.Infof("Toleration does not exist on deployment, add it")
		if existingTolerations == nil {
			glog.Infof("Tolerations do not exist, patching empty tolerations")
			patch = append(patch, patchOperation{
				Op:    "add",
				Path:  "/spec/template/spec/tolerations",
				Value: []string{},
			})
		}
		// This is appended only after we determine that the
		// toleration exists
		patch = append(patch, patchOperation{
			Op:   "add",
			Path: "/spec/template/spec/tolerations/-",
			Value: map[string]string{
				"key":      "spot",
				"operator": "Equal",
				"value":    `true`,
				"effect":   "NoSchedule",
			},
		})
	}
	return patch
}

func updateNodeSelector(existingNodeSelector map[string]string) (patch []patchOperation) {
	if existingNodeSelector == nil {
		glog.Infof("No node selector defined, add")
		patch = append(patch, patchOperation{
			Op:   "add",
			Path: "/spec/template/spec/nodeSelector",
			Value: map[string]string{
				"spot": "true",
			},
		})
	} else {
		glog.Infof("node selector defined, append")
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  "/spec/template/spec/nodeSelector/spot",
			Value: "true",
		})
	}
	return patch
}

func createPatch(existingTolerations []corev1.Toleration, existingNodeSelector map[string]string) ([]byte, error) {
	var patch []patchOperation

	if !tolerationAlreadyExists(existingTolerations) {
		glog.Infof("Patching resource to add toleration")
		patch = append(patch, updateTolerations(existingTolerations)...)
	}

	if !selectorAlreadyExists(existingNodeSelector) {
		glog.Infof("Patching resource to add node selector")
		patch = append(patch, updateNodeSelector(existingNodeSelector)...)
	}

	// if no patches were made, return a nil patch
	if patch == nil {
		return nil, nil
	}

	return json.Marshal(patch)
}

// main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var (
		existingTolerations  []corev1.Toleration
		existingNodeSelector map[string]string
		//objectMeta                      *metav1.ObjectMeta
		resourceName string
		//mutateRequied                   bool
	)

	glog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

	// First thing we should do is check the namespace of the request because if it's in a namespace
	// we ignore, don't need to advance any further anyways
	if isNameSpaceIgnored(ignoredNamespaces, req.Namespace) {
		glog.Infof("Skip mutation for %v for because it's in an ignored namespace:%v", req.Name, req.Namespace)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	//mutateRequied = false

	switch req.Kind.Kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
			glog.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		// These are set here because in the event that we add another case for daemonset, the values are
		// dereferenced from their k8s "kind" and it can be treated generically going forward
		//resourceName, resourceNamespace, objectMeta = deployment.Name, deployment.Namespace, &deployment.ObjectMeta
		existingTolerations = deployment.Spec.Template.Spec.Tolerations
		existingNodeSelector = deployment.Spec.Template.Spec.NodeSelector

		//mutateRequied = true
	// We should never hit this, since we are only sending deployments through from the webhook config,
	// but just incase, let's handle if a resource type gets through that isn't currently supported
	default:
		glog.Infof("%v is a %v, is not a supported resource type for the webhook", req.Name, req.Kind.Kind)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	glog.Infof("Tolerations: %v", existingTolerations)
	glog.Infof("Node Selector: %v", existingNodeSelector)

	// At this point we know it's a resource kind we support and it's not in an ingored namespace,
	// let's check if the tolerations are already set, and if not, set them

	//TODO:  Here, we should check to see if the deployment already has the required node selector and
	//       tolerations, if it does, then we shouldn't need to do anything
	// if !mutationRequired(ignoredNamespaces, objectMeta, existingTolerations, existingNodeSelector) || mutateRequied == false {
	// 	glog.Infof("Skipping validation for %s/%s due to policy check", resourceNamespace, resourceName)
	// 	return &v1beta1.AdmissionResponse{
	// 		Allowed: true,
	// 	}
	// }

	patchBytes, err := createPatch(existingTolerations, existingNodeSelector)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	// We didn't apply a patch, so let's just return with no modifications
	if patchBytes == nil {
		glog.Infof("No Changes were required")
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	glog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		glog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	// Pull in the environment variable for the namespaces we should ignore and add it to the
	// list of ingored namespaces
	envNamespaceList, set := os.LookupEnv("IGNORED_NAMESPACES")

	if set {
		ignoredNamespaces = append(ignoredNamespaces, strings.Split(envNamespaceList, ";")...)
	}
	glog.Infof("The following namspaces are excluded from spot instance injections: %v", ignoredNamespaces)

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		fmt.Println(r.URL.Path)
		if r.URL.Path == "/mutate" {
			admissionResponse = whsvr.mutate(&ar)
		}
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		glog.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	glog.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
