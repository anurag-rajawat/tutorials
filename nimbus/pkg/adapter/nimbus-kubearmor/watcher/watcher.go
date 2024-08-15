package watcher

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/anurag-rajawat/tutorials/nimbus/pkg/adapter/k8s"
)

type Watcher struct {
	dynamicClient dynamic.Interface
	factory       dynamicinformer.DynamicSharedInformerFactory
	logger        logr.Logger
}

func (w *Watcher) isOrphan(ctx context.Context, ownerReferences []metav1.OwnerReference, owner string, namespace string) bool {
	if len(ownerReferences) == 0 {
		return false
	}
	for _, ownerReference := range ownerReferences {
		if ownerReference.Kind == owner {
			return !w.isOwnerExist(ctx, ownerReference, namespace)
		}
	}
	return true
}

func (w *Watcher) isOwnerExist(ctx context.Context, ownerReference metav1.OwnerReference, namespace string) bool {
	switch ownerReference.Kind {
	case "SecurityIntentBinding":
		sibGvr := schema.GroupVersionResource{
			Group:    "intent.security.nimbus.com",
			Version:  "v1alpha1",
			Resource: "securityintentbindings",
		}
		return w.isExist(ctx, sibGvr, ownerReference.Name, namespace)
	case "NimbusPolicy":
		npGvr := schema.GroupVersionResource{
			Group:    "intent.security.nimbus.com",
			Version:  "v1alpha1",
			Resource: "nimbuspolicies",
		}
		return w.isExist(ctx, npGvr, ownerReference.Name, namespace)
	default:
		return false
	}
}

func (w *Watcher) isExist(ctx context.Context, gvr schema.GroupVersionResource, name, namespace string) bool {
	_, err := w.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false
		}
		w.logger.Error(err, "failed to get Policy", "name", name, "namespace", namespace)
		return false
	}
	return true
}

func (w *Watcher) decodePolicy(obj *unstructured.Unstructured, into client.Object) {
	bytes, err := obj.MarshalJSON()
	if err != nil {
		w.logger.Error(err, "failed to marshal policy", "name", obj.GetName(), "namespace", obj.GetNamespace())
		return
	}

	decoder := k8sscheme.Codecs.UniversalDeserializer()
	_, _, err = decoder.Decode(bytes, nil, into)
	if err != nil {
		w.logger.Error(err, "failed to decode policy", "name", obj.GetName(), "namespace", obj.GetNamespace())
		return
	}

	return
}

func NewWatcher(ctx context.Context) *Watcher {
	logger := log.FromContext(ctx)
	dynamicClient, err := k8s.NewDynamicClient()
	if err != nil {
		logger.Error(err, "failed to create kubernetes client")
		return nil
	}
	return &Watcher{
		dynamicClient: dynamicClient,
		logger:        logger,
		factory:       dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute),
	}
}
