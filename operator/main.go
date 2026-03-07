package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	storagev1 "github.com/kubenas/kubenas/operator/api/v1"
	storagev1alpha1 "github.com/kubenas/kubenas/operator/api/v1alpha1"
	"github.com/kubenas/kubenas/operator/controllers"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	utilruntime.Must(storagev1alpha1.AddToScheme(scheme))
	utilruntime.Must(storagev1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address for metrics endpoint.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address for health probe endpoint.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for HA deployments.")
	flag.Parse()

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "kubenas-operator-leader",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize node agent client.
	// In production this connects to the DaemonSet pods via the K8s API.
	agentClient := controllers.NewKubernetesAgentClient(mgr.GetClient())

	// Register all controllers.
	if err = (&controllers.DiskReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Disk"),
		Scheme: mgr.GetScheme(),
		Agent:  agentClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Disk")
		os.Exit(1)
	}

	if err = (&controllers.ArrayReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Array"),
		Scheme: mgr.GetScheme(),
		Agent:  agentClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Array")
		os.Exit(1)
	}

	if err = (&controllers.PoolReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Pool"),
		Scheme: mgr.GetScheme(),
		Agent:  agentClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pool")
		os.Exit(1)
	}

	if err = (&controllers.ShareReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Share"),
		Scheme: mgr.GetScheme(),
		Agent:  agentClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Share")
		os.Exit(1)
	}

	if err = (&controllers.ParityReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Parity"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Parity")
		os.Exit(1)
	}

	if err = (&controllers.FailureReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Failure"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Failure")
		os.Exit(1)
	}

	if err = controllers.NewDiskController(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("DiskV1"), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DiskV1")
		os.Exit(1)
	}
	if err = controllers.NewDiskClaimController(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("DiskClaimV1"), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DiskClaimV1")
		os.Exit(1)
	}
	if err = controllers.NewUnassignedDiskController(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("UnassignedDiskV1"), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "UnassignedDiskV1")
		os.Exit(1)
	}
	if err = controllers.NewPoolController(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("StoragePoolV1"), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StoragePoolV1")
		os.Exit(1)
	}
	if err = controllers.NewFilesystemController(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("FilesystemV1"), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FilesystemV1")
		os.Exit(1)
	}
	if err = controllers.NewShareControllerV1(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("ShareV1"), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ShareV1")
		os.Exit(1)
	}
	if err = controllers.NewVolumeController(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("VolumeV1"), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VolumeV1")
		os.Exit(1)
	}
	if err = controllers.NewTierController(mgr.GetClient(), ctrl.Log.WithName("controllers").WithName("TierPolicyV1"), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TierPolicyV1")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting KubeNAS operator")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
