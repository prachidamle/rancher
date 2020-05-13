package nsserviceaccount

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rancher/rancher/pkg/settings"
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type namespaceSvcAccountHandler struct {
	clusterCtx *config.UserContext
}

func Register(ctx context.Context, cluster *config.UserContext) {
	logrus.Debugf("Registering namespaceHandler for checking default serviceaccount")
	nsh := &namespaceSvcAccountHandler{
		clusterCtx: cluster,
	}
	cluster.Core.Namespaces("").AddHandler(ctx, "namespaceHandler", nsh.Sync)
}

func (nsh *namespaceSvcAccountHandler) Sync(key string, ns *corev1.Namespace) (runtime.Object, error) {
	if ns == nil {
		return nil, nil
	}
	logrus.Debugf("namespaceSvcAccountHandler: Sync: key=%v, ns=%+v", key, *ns)

	//handle default svcAccount of system namespaces
	err := nsh.handleSystemNS(key)
	if err != nil {
		logrus.Errorf("namespaceSvcAccountHandler: Sync: error handling default ServiceAccount of namespace key=%v, err=%v", key, err)
		return nil, nil
	}
	return nil, nil
}

func (nsh *namespaceSvcAccountHandler) handleSystemNS(namespace string) error {

	if (namespace != "kube-system" && namespace != "default") && nsh.isSystemNS(namespace) {
		var svcAccounts clientv1.ServiceAccountInterface
		svcAccounts = nsh.clusterCtx.K8sClient.CoreV1().ServiceAccounts(namespace)

		defSvcAccnt, err := svcAccounts.Get("default", metav1.GetOptions{})
		if err != nil {
			logrus.Errorf("namespaceSvcAccountHandler: error listing serviceaccount flag: Sync: key=%v, err=%+v", namespace, err)
			return err
		}

		if defSvcAccnt.AutomountServiceAccountToken == nil || *defSvcAccnt.AutomountServiceAccountToken == true {
			automountServiceAccountToken := false
			defSvcAccnt.AutomountServiceAccountToken = &automountServiceAccountToken
			defSvcAccnt.Namespace = namespace
			logrus.Debugf("namespaceSvcAccountHandler: updating default serviceaccount key=%v", defSvcAccnt)
			_, err = svcAccounts.Update(defSvcAccnt)
			if err != nil {
				logrus.Errorf("namespaceSvcAccountHandler: error updating serviceaccnt flag: Sync: key=%v, err=%+v", namespace, err)
				return err
			}
		}

	}

	return nil
}

func (nsh *namespaceSvcAccountHandler) isSystemNS(namespace string) bool {
	systemNamespacesStr := settings.SystemNamespaces.Get()
	systemNamespaces := make(map[string]bool)

	splitted := strings.Split(systemNamespacesStr, ",")
	for _, s := range splitted {
		ns := strings.TrimSpace(s)
		systemNamespaces[ns] = true
	}

	return systemNamespaces[namespace]
}
