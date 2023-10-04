package watcher

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	ctrlevent "sigs.k8s.io/controller-runtime/pkg/event"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
)

var natsGVK = schema.GroupVersionResource{
	Group:    natsv1alpha1.GroupVersion.Group,
	Version:  natsv1alpha1.GroupVersion.Version,
	Resource: "nats",
}

type NatsWatcher struct {
	dynamicClient  *dynamic.DynamicClient
	namespace      string
	NatsCREventsCh chan ctrlevent.GenericEvent
	watcher        watch.Interface
	stopped        bool
}

func NewWatcher(dynamicClient *dynamic.DynamicClient, namespace string) *NatsWatcher {
	return &NatsWatcher{
		dynamicClient:  dynamicClient,
		namespace:      namespace,
		NatsCREventsCh: make(chan ctrlevent.GenericEvent),
	}
}

func (w *NatsWatcher) Start() error {
	var err error
	w.watcher, err = w.dynamicClient.Resource(natsGVK).Namespace(w.namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch NATS CRs: %v", err)
	}

	w.stopped = false
	go func() {
		for {
			event, ok := <-w.watcher.ResultChan()
			if !ok {
				if w.stopped {
					log.Println("watcher stopped")
					return
				}
				log.Println("unexpected error occurred")
				w.NatsCREventsCh <- ctrlevent.GenericEvent{}
				return
			}

			log.Println("event received:", event.Type)
			w.NatsCREventsCh <- ctrlevent.GenericEvent{
				Object: &natsv1alpha1.NATS{
					TypeMeta: metav1.TypeMeta{
						Kind:       natsGVK.Resource,
						APIVersion: natsGVK.Version,
					},
				},
			}
		}
	}()

	return nil
}

func (w *NatsWatcher) Stop() {
	w.watcher.Stop()
	w.stopped = true
}
