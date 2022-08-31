package controllers

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func debug(log logr.Logger, msg string, rawOBJ interface{}, keysAndValues ...interface{})  {
	if obj, ok := rawOBJ.(client.Object); ok {
		kvs := append([]interface{}{"namespace", obj.GetNamespace(), "name", obj.GetName()}, keysAndValues...)
		log.V(logging.InfoLevel).Info(msg.kvs...)
	}
	
}