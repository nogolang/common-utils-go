package kratosUtils

import (
	"log"
	"path/filepath"

	kuberegistry "github.com/go-kratos/kratos/contrib/registry/kubernetes/v2"
	"github.com/nogolang/common-utils-go/configUtils"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func NewKratosRegisterK8s(allConfig *configUtils.CommonConfig) *kuberegistry.Registry {
	//未开启注册时，返回一个空的即可，反正我们也不会去用
	if allConfig.K8s == nil || !allConfig.K8s.Register {
		return &kuberegistry.Registry{}
	}

	//如果运行在pod里，那么会注入名称空间
	//直接使用kuberegistry.GetNamespace()即可
	//nameSpace := os.Getenv("POD_NAMESPACE")

	set, err := getClientSet()
	if err != nil {
		log.Fatal("NewKratosRegisterK8s", zap.Error(err))
		return nil
	}
	reg := kuberegistry.NewRegistry(set, kuberegistry.GetNamespace())
	//开启start，才能监听pod变化
	reg.Start()
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
