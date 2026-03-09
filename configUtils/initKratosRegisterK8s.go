package configUtils

import (
	"path/filepath"

	kuberegistry "github.com/go-kratos/kratos/contrib/registry/kubernetes/v2"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func NewKratosRegisterK8s(logger *zap.Logger, allConfig *CommonConfig) *kuberegistry.Registry {
	//如果是dev，返回一个空的即可，反正我们也不会去用
	if IsDev() {
		return &kuberegistry.Registry{}
	}

	//如果运行在pod里，那么会注入名称空间
	//直接使用kuberegistry.GetNamespace()即可
	//nameSpace := os.Getenv("POD_NAMESPACE")

	set, err := getClientSet()
	if err != nil {
		logger.Fatal("NewKratosRegisterK8s", zap.Error(err))
		return nil
	}
	reg := kuberegistry.NewRegistry(set, kuberegistry.GetNamespace())
	return reg
}
func getClientSet() (*kubernetes.Clientset, error) {
	restConfig, err := rest.InClusterConfig()
	home := homedir.HomeDir()

	if err != nil {
		kubeconfig := filepath.Join(home, ".kube", "config")
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return clientSet, nil
}
