package main

import (
	"context"
	"flag"
	"os"
	
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/golang/glog"
)

var cmd = &cobra.Command{
	Use:          "kubedel",
	SilenceUsage: true,
	Run: func(c *cobra.Command, args []string) {
		if err := exec(args); err != nil {
			glog.Errorf("%v", err)
			os.Exit(1)
		}
	},
}

var kubeconfig string
var args []string

func init() {
	args = []string{}

	viper.AutomaticEnv()
	// parse the go default flagset to get flags for glog and other packages in future
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// defaulting this to true so that logs are printed to console
	flag.Set("logtostderr", "true")

	cmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", kubeconfig, "path to kubeconfig")

	cmd.PersistentFlags().MarkHidden("alsologtostderr")
	cmd.PersistentFlags().MarkHidden("log_backtrace_at")
	cmd.PersistentFlags().MarkHidden("log_dir")
	cmd.PersistentFlags().MarkHidden("logtostderr")
	cmd.PersistentFlags().MarkHidden("master")
	cmd.PersistentFlags().MarkHidden("stderrthreshold")
	cmd.PersistentFlags().MarkHidden("vmodule")

	// suppress the incorrect prefix in glog output
	flag.CommandLine.Parse([]string{})
	viper.BindPFlags(cmd.PersistentFlags())
}

func main() {
	cmd.Execute()
}

func exec(args []string) error {
	kubeConfig := viper.GetString("kubeconfig")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			glog.Errorf("could not find client configuration: %v", err)
			return err
		}
		glog.V(2).Infof("obtained client config successfully")
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Errorf("could not initialize kubeclient: %v", err)
		return err
	}
	for _, ns := range args {
		glog.V(2).Infof("fetching ns: %v", ns)
		nsObj, err := kubeClient.CoreV1().Namespaces().Get(context.Background(), ns, metav1.GetOptions{})
		if err != nil {
			glog.Errorf("error fetching ns: %v", err)
			continue
		}
		nsObj.Spec.Finalizers = []corev1.FinalizerName{}
		glog.V(2).Infof("finalizers: %+v", nsObj.Spec.Finalizers)
		glog.V(2).Infof("finalizing ns: %v", ns)
		if _, err := kubeClient.CoreV1().Namespaces().Finalize(context.Background(), nsObj, metav1.UpdateOptions{}); err != nil {
			glog.Errorf("error deleting finalizer for ns: %v", err)
			continue
		}

		glog.V(2).Infof("deleting ns: %v", ns)
		if err := kubeClient.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{}); err != nil {
			glog.Errorf("error deleting ns: %v", err)
			continue
		}		
	}
	return nil
}
