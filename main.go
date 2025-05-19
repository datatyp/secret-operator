package main

import (
	"context"
	"os"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	secretNamePattern = regexp.MustCompile(`^([a-z0-9-]+)--([a-z0-9-]+)$`)
)

type SecretReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	SourceNamespace string
}

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if req.Namespace != r.SourceNamespace {
		return ctrl.Result{}, nil
	}
	logger := log.FromContext(ctx)

	var src corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &src); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	matches := secretNamePattern.FindStringSubmatch(src.Name)
	if len(matches) != 3 {
		return ctrl.Result{}, nil
	}
	targetNS, targetName := matches[1], matches[2]

	dst := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: targetNS,
			Name:      src.Name,
		},
	}
	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, &dst, func() error {
		dst.Data = src.Data
		dst.Immutable = src.Immutable
		dst.StringData = nil
		dst.Type = src.Type
		dst.Labels = src.Labels
		dst.Annotations = src.Annotations
		return nil
	})
	if err != nil {
		logger.Error(err, "copy failed", "namespace", targetNS, "name", targetName)
		return ctrl.Result{}, err
	}
	logger.Info("copied secret", "operation", op, "namespace", targetNS, "name", targetName)

	return ctrl.Result{}, nil
}

func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Complete(r)
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	sourceNS := os.Getenv("SOURCE_NAMESPACE")
	if sourceNS == "" {
		sourceNS = "kafka"
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		panic(err)
	}

	r := &SecretReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		SourceNamespace: sourceNS,
	}
	if err := r.SetupWithManager(mgr); err != nil {
		panic(err)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		panic(err)
	}
}

