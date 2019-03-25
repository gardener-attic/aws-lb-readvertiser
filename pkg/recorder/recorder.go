package recorder

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/recorder"
)

type provider struct {
	// scheme to specify when creating a recorder
	// eventBroadcaster to create new recorder instance
	eventBroadcaster record.EventBroadcaster
	// logger is the logger to use when logging diagnostic event info
	logger logr.Logger
}

// NewProvider create a new Provider instance.
func NewProvider(client typedcorev1.EventInterface, logger logr.Logger) recorder.Provider {
	p := &provider{logger: logger}
	p.eventBroadcaster = record.NewBroadcaster()
	p.eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: client})
	p.eventBroadcaster.StartEventWatcher(
		func(e *corev1.Event) {
			p.logger.V(1).Info(e.Type, "object", e.InvolvedObject, "reason", e.Reason, "message", e.Message)
		})

	return p
}

func (p *provider) GetEventRecorderFor(name string) record.EventRecorder {
	return p.eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: name})
}
