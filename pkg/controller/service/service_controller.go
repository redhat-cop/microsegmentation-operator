package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_service")

const annotationBase = "microsegmentation-operator.redhat-cop.io"
const microsgmentationAnnotation = annotationBase + "/microsegmentation"
const additionalInboundPortsAnnotation = annotationBase + "/additional-inbound-ports"
const inboundPodLabels = annotationBase + "/inbound-pod-labels"
const inboundNamespaceLables = annotationBase + "/inbound-namespace-labels"
const outboundPodLabels = annotationBase + "/outbound-pod-labels"
const outboundNamespaceLabels = annotationBase + "/outbound-namespace-labels"
const outboundPorts = annotationBase + "/outbound-ports"
const controllerName = "service-controller"

// Add creates a new Service Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileService{
		ReconcilerBase: util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetRecorder(controllerName)),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	isAnnotatedService := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			_, ok := e.ObjectOld.(*corev1.Service)
			if !ok {
				return false
			}
			_, ok = e.ObjectNew.(*corev1.Service)
			if !ok {
				return false
			}
			oldValue, _ := e.MetaOld.GetAnnotations()[microsgmentationAnnotation]
			newValue, _ := e.MetaNew.GetAnnotations()[microsgmentationAnnotation]
			old := oldValue == "true"
			new := newValue == "true"
			return old != new
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Object.(*corev1.Service)
			if !ok {
				return false
			}
			value, _ := e.Meta.GetAnnotations()[microsgmentationAnnotation]
			return value == "true"
		},
	}

	// Watch for changes to primary resource Service
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}, isAnnotatedService)
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Service
	err = c.Watch(&source.Kind{Type: &networking.NetworkPolicy{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &corev1.Service{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileService{}

// ReconcileService reconciles a Service object
type ReconcileService struct {
	util.ReconcilerBase
}

// Reconcile reads that state of the cluster for a Service object and makes changes based on the state read
// and what is in the Service.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Service")

	// Fetch the Service instance
	instance := &corev1.Service{}
	err := r.GetClient().Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Pod object
	networkPolicy := getNetworkPolicy(instance)

	if instance.Annotations[microsgmentationAnnotation] == "true" {
		err = r.CreateOrUpdateResource(instance, instance.GetNamespace(), networkPolicy)
		if err != nil {
			log.Error(err, "unable to create NetworkPolicy", "NetworkPolicy", networkPolicy)
			return r.manageError(err, instance)
		}
	} else {
		err = r.DeleteResource(networkPolicy)
		if err != nil {
			log.Error(err, "unable to delete NetworkPolicy", "NetworkPolicy", networkPolicy)
			return r.manageError(err, instance)
		}
	}

	return reconcile.Result{}, nil
}

func getNetworkPolicy(service *corev1.Service) *networking.NetworkPolicy {
	networkPolicy := &networking.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.GetName(),
			Namespace: service.GetNamespace(),
		},
		Spec: networking.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: service.Spec.Selector,
			},
			Egress:  []networking.NetworkPolicyEgressRule{},
			Ingress: []networking.NetworkPolicyIngressRule{},
		},
	}
	_, ok1 := service.Annotations[inboundPodLabels]
	_, ok2 := service.Annotations[inboundNamespaceLables]

	if !ok1 && !ok2 {
		//if there are no pod or namespaces annotations we create one for inbound traffic only from the current namespace
		networkPolicyIngressRule := networking.NetworkPolicyIngressRule{
			From: []networking.NetworkPolicyPeer{networking.NetworkPolicyPeer{
				PodSelector: &metav1.LabelSelector{},
			}},
			Ports: append(getPortsFromService(service.Spec.Ports), getPortsFromAnnotation(service.Annotations[additionalInboundPortsAnnotation])...),
		}
		networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, networkPolicyIngressRule)
	}

	if inboundPodLabels, ok := service.Annotations[inboundPodLabels]; ok {
		networkPolicyIngressRule := networking.NetworkPolicyIngressRule{
			From: []networking.NetworkPolicyPeer{networking.NetworkPolicyPeer{
				PodSelector: getLabelSelectorFromAnnotation(inboundPodLabels),
			}},
			Ports: append(getPortsFromService(service.Spec.Ports), getPortsFromAnnotation(service.Annotations[additionalInboundPortsAnnotation])...),
		}
		networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, networkPolicyIngressRule)

	}
	if inboundNamespaceLables, ok := service.Annotations[inboundNamespaceLables]; ok {
		networkPolicyIngressRule := networking.NetworkPolicyIngressRule{
			From: []networking.NetworkPolicyPeer{networking.NetworkPolicyPeer{
				NamespaceSelector: getLabelSelectorFromAnnotation(inboundNamespaceLables),
			}},
			Ports: append(getPortsFromService(service.Spec.Ports), getPortsFromAnnotation(service.Annotations[additionalInboundPortsAnnotation])...),
		}
		networkPolicy.Spec.Ingress = append(networkPolicy.Spec.Ingress, networkPolicyIngressRule)

	}
	if outboundPodLabels, ok := service.Annotations[outboundPodLabels]; ok {
		networkPolicyEgressRule := networking.NetworkPolicyEgressRule{
			To: []networking.NetworkPolicyPeer{networking.NetworkPolicyPeer{
				PodSelector: getLabelSelectorFromAnnotation(outboundPodLabels),
			}},
			Ports: getPortsFromAnnotation(service.Annotations[outboundPorts]),
		}
		networkPolicy.Spec.Egress = append(networkPolicy.Spec.Egress, networkPolicyEgressRule)
	}
	if outboundNamespaceLabels, ok := service.Annotations[outboundNamespaceLabels]; ok {
		networkPolicyEgressRule := networking.NetworkPolicyEgressRule{
			To: []networking.NetworkPolicyPeer{networking.NetworkPolicyPeer{
				NamespaceSelector: getLabelSelectorFromAnnotation(outboundNamespaceLabels),
			}},
			Ports: getPortsFromAnnotation(service.Annotations[outboundPorts]),
		}
		networkPolicy.Spec.Egress = append(networkPolicy.Spec.Egress, networkPolicyEgressRule)
	}

	return networkPolicy
}

func getPortsFromService(ports []corev1.ServicePort) []networking.NetworkPolicyPort {
	networkPolicyPorts := []networking.NetworkPolicyPort{}
	for _, port := range ports {
		iport := port.TargetPort
		iprotocol := port.Protocol
		networkPolicyPorts = append(networkPolicyPorts, networking.NetworkPolicyPort{
			Port:     &iport,
			Protocol: &iprotocol,
		})
	}
	return networkPolicyPorts
}

func getPortsFromAnnotation(ports string) []networking.NetworkPolicyPort {
	// tihs annotation looks like this: 9999/TCP,8888/UDP
	networkPolicyPorts := []networking.NetworkPolicyPort{}
	if ports == "" {
		return networkPolicyPorts
	}
	portsStrings := strings.Split(ports, ",")
	for _, portString := range portsStrings {
		intport, err := strconv.Atoi(portString[:strings.Index(portString, "/")])
		if err != nil {
			log.Error(err, "unable to convert port to integer", "port", portString[:strings.Index(portString, "/")])
			continue
		}
		port := intstr.FromInt(intport)
		protocol := corev1.Protocol(strings.ToUpper(portString[strings.Index(portString, "/")+1:]))
		networkPolicyPorts = append(networkPolicyPorts, networking.NetworkPolicyPort{
			Port:     &port,
			Protocol: &protocol,
		})
	}
	return networkPolicyPorts
}

func getLabelSelectorFromAnnotation(labels string) *metav1.LabelSelector {
	// tihs annotation looks like this: label1=value,label2=value2
	labelMap := map[string]string{}
	labelsStrings := strings.Split(labels, ",")
	for _, labelString := range labelsStrings {
		label := labelString[:strings.Index(labelString, "=")]
		value := labelString[strings.Index(labelString, "=")+1:]
		labelMap[label] = value
	}
	return &metav1.LabelSelector{
		MatchLabels: labelMap,
	}
}

func (r *ReconcileService) manageError(issue error, instance runtime.Object) (reconcile.Result, error) {
	r.GetRecorder().Event(instance, "Warning", "ProcessingError", issue.Error())
	return reconcile.Result{
		RequeueAfter: time.Minute * 2,
		Requeue:      true,
	}, nil
}
