package catalog

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// PodDisruptionBudgetClient provides methods for managing PDBs
type PodDisruptionBudgetClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewPodDisruptionBudgetClient creates a new PDB client
func NewPodDisruptionBudgetClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *PodDisruptionBudgetClient {
	return &PodDisruptionBudgetClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a PDB by name
func (c *PodDisruptionBudgetClient) Get(namespace, name string) (*policyv1.PodDisruptionBudget, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists PDBs in a namespace
func (c *PodDisruptionBudgetClient) List(namespace string, labelSelector string) ([]*policyv1.PodDisruptionBudget, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	pdbs, err := c.clientset.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*policyv1.PodDisruptionBudget, len(pdbs.Items))
	for i := range pdbs.Items {
		result[i] = &pdbs.Items[i]
	}

	return result, nil
}

// HorizontalPodAutoscalerClient provides methods for managing HPAs
type HorizontalPodAutoscalerClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewHorizontalPodAutoscalerClient creates a new HPA client
func NewHorizontalPodAutoscalerClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *HorizontalPodAutoscalerClient {
	return &HorizontalPodAutoscalerClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns an HPA by name
func (c *HorizontalPodAutoscalerClient) Get(namespace, name string) (*autoscalingv1.HorizontalPodAutoscaler, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists HPAs in a namespace
func (c *HorizontalPodAutoscalerClient) List(namespace string, labelSelector string) ([]*autoscalingv1.HorizontalPodAutoscaler, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	hpas, err := c.clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*autoscalingv1.HorizontalPodAutoscaler, len(hpas.Items))
	for i := range hpas.Items {
		result[i] = &hpas.Items[i]
	}

	return result, nil
}

// CronJobClient provides methods for managing CronJobs
type CronJobClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewCronJobClient creates a new CronJob client
func NewCronJobClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *CronJobClient {
	return &CronJobClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a CronJob by name
func (c *CronJobClient) Get(namespace, name string) (*batchv1.CronJob, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists CronJobs in a namespace
func (c *CronJobClient) List(namespace string, labelSelector string) ([]*batchv1.CronJob, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	cronjobs, err := c.clientset.BatchV1().CronJobs(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*batchv1.CronJob, len(cronjobs.Items))
	for i := range cronjobs.Items {
		result[i] = &cronjobs.Items[i]
	}

	return result, nil
}

// JobClient provides methods for managing Jobs
type JobClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewJobClient creates a new Job client
func NewJobClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *JobClient {
	return &JobClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a Job by name
func (c *JobClient) Get(namespace, name string) (*batchv1.Job, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists Jobs in a namespace
func (c *JobClient) List(namespace string, labelSelector string) ([]*batchv1.Job, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	jobs, err := c.clientset.BatchV1().Jobs(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*batchv1.Job, len(jobs.Items))
	for i := range jobs.Items {
		result[i] = &jobs.Items[i]
	}

	return result, nil
}

// StorageClassClient provides methods for managing StorageClasses
type StorageClassClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewStorageClassClient creates a new StorageClass client
func NewStorageClassClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *StorageClassClient {
	return &StorageClassClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a StorageClass by name
func (c *StorageClassClient) Get(name string) (*storagev1.StorageClass, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
}

// List lists all StorageClasses
func (c *StorageClassClient) List() ([]*storagev1.StorageClass, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	storageClasses, err := c.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]*storagev1.StorageClass, len(storageClasses.Items))
	for i := range storageClasses.Items {
		result[i] = &storageClasses.Items[i]
	}

	return result, nil
}

// PersistentVolumeClient provides methods for managing PersistentVolumes
type PersistentVolumeClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewPersistentVolumeClient creates a new PersistentVolume client
func NewPersistentVolumeClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *PersistentVolumeClient {
	return &PersistentVolumeClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a PersistentVolume by name
func (c *PersistentVolumeClient) Get(name string) (*corev1.PersistentVolume, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
}

// List lists all PersistentVolumes
func (c *PersistentVolumeClient) List(labelSelector string) ([]*corev1.PersistentVolume, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	pvs, err := c.clientset.CoreV1().PersistentVolumes().List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.PersistentVolume, len(pvs.Items))
	for i := range pvs.Items {
		result[i] = &pvs.Items[i]
	}

	return result, nil
}

// ResourceQuotaClient provides methods for managing ResourceQuotas
type ResourceQuotaClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewResourceQuotaClient creates a new ResourceQuota client
func NewResourceQuotaClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *ResourceQuotaClient {
	return &ResourceQuotaClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a ResourceQuota by name
func (c *ResourceQuotaClient) Get(namespace, name string) (*corev1.ResourceQuota, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists ResourceQuotas in a namespace
func (c *ResourceQuotaClient) List(namespace string, labelSelector string) ([]*corev1.ResourceQuota, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	quotas, err := c.clientset.CoreV1().ResourceQuotas(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.ResourceQuota, len(quotas.Items))
	for i := range quotas.Items {
		result[i] = &quotas.Items[i]
	}

	return result, nil
}

// LimitRangeClient provides methods for managing LimitRanges
type LimitRangeClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewLimitRangeClient creates a new LimitRange client
func NewLimitRangeClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *LimitRangeClient {
	return &LimitRangeClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a LimitRange by name
func (c *LimitRangeClient) Get(namespace, name string) (*corev1.LimitRange, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists LimitRanges in a namespace
func (c *LimitRangeClient) List(namespace string, labelSelector string) ([]*corev1.LimitRange, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	limitRanges, err := c.clientset.CoreV1().LimitRanges(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.LimitRange, len(limitRanges.Items))
	for i := range limitRanges.Items {
		result[i] = &limitRanges.Items[i]
	}

	return result, nil
}

// PodTemplateClient provides methods for managing PodTemplates
type PodTemplateClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewPodTemplateClient creates a new PodTemplate client
func NewPodTemplateClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *PodTemplateClient {
	return &PodTemplateClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a PodTemplate by name
func (c *PodTemplateClient) Get(namespace, name string) (*corev1.PodTemplate, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().PodTemplates(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists PodTemplates in a namespace
func (c *PodTemplateClient) List(namespace string, labelSelector string) ([]*corev1.PodTemplate, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	templates, err := c.clientset.CoreV1().PodTemplates(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.PodTemplate, len(templates.Items))
	for i := range templates.Items {
		result[i] = &templates.Items[i]
	}

	return result, nil
}

// ReplicationControllerClient provides methods for managing ReplicationControllers
type ReplicationControllerClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewReplicationControllerClient creates a new ReplicationController client
func NewReplicationControllerClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *ReplicationControllerClient {
	return &ReplicationControllerClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a ReplicationController by name
func (c *ReplicationControllerClient) Get(namespace, name string) (*corev1.ReplicationController, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().ReplicationControllers(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists ReplicationControllers in a namespace
func (c *ReplicationControllerClient) List(namespace string, labelSelector string) ([]*corev1.ReplicationController, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	controllers, err := c.clientset.CoreV1().ReplicationControllers(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.ReplicationController, len(controllers.Items))
	for i := range controllers.Items {
		result[i] = &controllers.Items[i]
	}

	return result, nil
}

// EndpointClient provides methods for managing Endpoints
type EndpointClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewEndpointClient creates a new Endpoint client
func NewEndpointClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *EndpointClient {
	return &EndpointClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns an Endpoint by name
func (c *EndpointClient) Get(namespace, name string) (*corev1.Endpoints, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists Endpoints in a namespace
func (c *EndpointClient) List(namespace string, labelSelector string) ([]*corev1.Endpoints, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	endpoints, err := c.clientset.CoreV1().Endpoints(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.Endpoints, len(endpoints.Items))
	for i := range endpoints.Items {
		result[i] = &endpoints.Items[i]
	}

	return result, nil
}

// EventClient provides methods for managing Events
type EventClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewEventClient creates a new Event client
func NewEventClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *EventClient {
	return &EventClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns an Event by name
func (c *EventClient) Get(namespace, name string) (*corev1.Event, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().Events(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists Events in a namespace
func (c *EventClient) List(namespace string, labelSelector string) ([]*corev1.Event, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	events, err := c.clientset.CoreV1().Events(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.Event, len(events.Items))
	for i := range events.Items {
		result[i] = &events.Items[i]
	}

	return result, nil
}

// ServiceAccountClient provides methods for managing ServiceAccounts
type ServiceAccountClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewServiceAccountClient creates a new ServiceAccount client
func NewServiceAccountClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *ServiceAccountClient {
	return &ServiceAccountClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a ServiceAccount by name
func (c *ServiceAccountClient) Get(namespace, name string) (*corev1.ServiceAccount, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists ServiceAccounts in a namespace
func (c *ServiceAccountClient) List(namespace string, labelSelector string) ([]*corev1.ServiceAccount, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	sas, err := c.clientset.CoreV1().ServiceAccounts(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.ServiceAccount, len(sas.Items))
	for i := range sas.Items {
		result[i] = &sas.Items[i]
	}

	return result, nil
}

// RoleClient provides methods for managing Roles
type RoleClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewRoleClient creates a new Role client
func NewRoleClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *RoleClient {
	return &RoleClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a Role by name
func (c *RoleClient) Get(namespace, name string) (*rbacv1.Role, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists Roles in a namespace
func (c *RoleClient) List(namespace string, labelSelector string) ([]*rbacv1.Role, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	roles, err := c.clientset.RbacV1().Roles(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.Role, len(roles.Items))
	for i := range roles.Items {
		result[i] = &roles.Items[i]
	}

	return result, nil
}

// ClusterRoleClient provides methods for managing ClusterRoles
type ClusterRoleClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewClusterRoleClient creates a new ClusterRole client
func NewClusterRoleClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *ClusterRoleClient {
	return &ClusterRoleClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a ClusterRole by name
func (c *ClusterRoleClient) Get(name string) (*rbacv1.ClusterRole, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
}

// List lists all ClusterRoles
func (c *ClusterRoleClient) List() ([]*rbacv1.ClusterRole, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	clusterRoles, err := c.clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.ClusterRole, len(clusterRoles.Items))
	for i := range clusterRoles.Items {
		result[i] = &clusterRoles.Items[i]
	}

	return result, nil
}

// RoleBindingClient provides methods for managing RoleBindings
type RoleBindingClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewRoleBindingClient creates a new RoleBinding client
func NewRoleBindingClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *RoleBindingClient {
	return &RoleBindingClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a RoleBinding by name
func (c *RoleBindingClient) Get(namespace, name string) (*rbacv1.RoleBinding, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists RoleBindings in a namespace
func (c *RoleBindingClient) List(namespace string, labelSelector string) ([]*rbacv1.RoleBinding, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	bindings, err := c.clientset.RbacV1().RoleBindings(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.RoleBinding, len(bindings.Items))
	for i := range bindings.Items {
		result[i] = &bindings.Items[i]
	}

	return result, nil
}

// ClusterRoleBindingClient provides methods for managing ClusterRoleBindings
type ClusterRoleBindingClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewClusterRoleBindingClient creates a new ClusterRoleBinding client
func NewClusterRoleBindingClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *ClusterRoleBindingClient {
	return &ClusterRoleBindingClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a ClusterRoleBinding by name
func (c *ClusterRoleBindingClient) Get(name string) (*rbacv1.ClusterRoleBinding, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
}

// List lists all ClusterRoleBindings
func (c *ClusterRoleBindingClient) List() ([]*rbacv1.ClusterRoleBinding, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	bindings, err := c.clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.ClusterRoleBinding, len(bindings.Items))
	for i := range bindings.Items {
		result[i] = &bindings.Items[i]
	}

	return result, nil
}

// PriorityClassClient provides methods for managing PriorityClasses
type PriorityClassClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewPriorityClassClient creates a new PriorityClass client
func NewPriorityClassClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *PriorityClassClient {
	return &PriorityClassClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a PriorityClass by name
func (c *PriorityClassClient) Get(name string) (*corev1.PriorityClass, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.CoreV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
}

// List lists all PriorityClasses
func (c *PriorityClassClient) List() ([]*corev1.PriorityClass, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	priorityClasses, err := c.clientset.CoreV1().PriorityClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]*corev1.PriorityClass, len(priorityClasses.Items))
	for i := range priorityClasses.Items {
		result[i] = &priorityClasses.Items[i]
	}

	return result, nil
}

// NetworkPolicyClient provides methods for managing NetworkPolicies
type NetworkPolicyClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewNetworkPolicyClient creates a new NetworkPolicy client
func NewNetworkPolicyClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *NetworkPolicyClient {
	return &NetworkPolicyClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a NetworkPolicy by name
func (c *NetworkPolicyClient) Get(namespace, name string) (*networkingv1.NetworkPolicy, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
}

// List lists NetworkPolicies in a namespace
func (c *NetworkPolicyClient) List(namespace string, labelSelector string) ([]*networkingv1.NetworkPolicy, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}

	policies, err := c.clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := make([]*networkingv1.NetworkPolicy, len(policies.Items))
	for i := range policies.Items {
		result[i] = &policies.Items[i]
	}

	return result, nil
}

// MutatingWebhookConfigurationClient provides methods for managing MutatingWebhookConfigurations
type MutatingWebhookConfigurationClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewMutatingWebhookConfigurationClient creates a new MutatingWebhookConfiguration client
func NewMutatingWebhookConfigurationClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *MutatingWebhookConfigurationClient {
	return &MutatingWebhookConfigurationClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a MutatingWebhookConfiguration by name
func (c *MutatingWebhookConfigurationClient) Get(name string) (*rbacv1.MutatingWebhookConfiguration, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
}

// List lists all MutatingWebhookConfigurations
func (c *MutatingWebhookConfigurationClient) List() ([]*rbacv1.MutatingWebhookConfiguration, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	whitelists, err := c.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.MutatingWebhookConfiguration, len(whitelists.Items))
	for i := range whitelists.Items {
		result[i] = &whitelists.Items[i]
	}

	return result, nil
}

// ValidatingWebhookConfigurationClient provides methods for managing ValidatingWebhookConfigurations
type ValidatingWebhookConfigurationClient struct {
	clientset *kubernetes.Clientset
	ctx       context.Context
	logger    *logrus.Logger
}

// NewValidatingWebhookConfigurationClient creates a new ValidatingWebhookConfiguration client
func NewValidatingWebhookConfigurationClient(clientset *kubernetes.Clientset, ctx context.Context, logger *logrus.Logger) *ValidatingWebhookConfigurationClient {
	return &ValidatingWebhookConfigurationClient{
		clientset: clientset,
		ctx:       ctx,
		logger:    logger,
	}
}

// Get returns a ValidatingWebhookConfiguration by name
func (c *ValidatingWebhookConfigurationClient) Get(name string) (*rbacv1.ValidatingWebhookConfiguration, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()
	return c.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
}

// List lists all ValidatingWebhookConfigurations
func (c *ValidatingWebhookConfigurationClient) List() ([]*rbacv1.ValidatingWebhookConfiguration, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	whitelists, err := c.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]*rbacv1.ValidatingWebhookConfiguration, len(whitelists.Items))
	for i := range whitelists.Items {
		result[i] = &whitelists.Items[i]
	}

	return result, nil
}

// ServiceDiscoveryClient wraps the KubernetesClient with additional discovery methods
type ServiceDiscoveryClient struct {
	client *KubernetesClient
}

// NewServiceDiscoveryClient creates a new ServiceDiscoveryClient
func NewServiceDiscoveryClient(client *KubernetesClient) *ServiceDiscoveryClient {
	return &ServiceDiscoveryClient{client: client}
}

// GetPodDisruptionBudgets returns all PDBs in a namespace
func (c *ServiceDiscoveryClient) GetPodDisruptionBudgets(namespace string) ([]*policyv1.PodDisruptionBudget, error) {
	return c.client.PolicyV1().PodDisruptionBudgets(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetHorizontalPodAutoscalers returns all HPAs in a namespace
func (c *ServiceDiscoveryClient) GetHorizontalPodAutoscalers(namespace string) ([]*autoscalingv1.HorizontalPodAutoscaler, error) {
	return c.client.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetCronJobs returns all CronJobs in a namespace
func (c *ServiceDiscoveryClient) GetCronJobs(namespace string) ([]*batchv1.CronJob, error) {
	return c.client.BatchV1().CronJobs(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetJobs returns all Jobs in a namespace
func (c *ServiceDiscoveryClient) GetJobs(namespace string) ([]*batchv1.Job, error) {
	return c.client.BatchV1().Jobs(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetStorageClasses returns all StorageClasses
func (c *ServiceDiscoveryClient) GetStorageClasses() ([]*storagev1.StorageClass, error) {
	return c.client.StorageV1().StorageClasses().List(c.ctx, metav1.ListOptions{})
}

// GetPersistentVolumes returns all PersistentVolumes
func (c *ServiceDiscoveryClient) GetPersistentVolumes() ([]*corev1.PersistentVolume, error) {
	return c.clientset.CoreV1().PersistentVolumes().List(c.ctx, metav1.ListOptions{})
}

// GetResourceQuotas returns all ResourceQuotas in a namespace
func (c *ServiceDiscoveryClient) GetResourceQuotas(namespace string) ([]*corev1.ResourceQuota, error) {
	return c.clientset.CoreV1().ResourceQuotas(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetLimitRanges returns all LimitRanges in a namespace
func (c *ServiceDiscoveryClient) GetLimitRanges(namespace string) ([]*corev1.LimitRange, error) {
	return c.clientset.CoreV1().LimitRanges(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetPodTemplates returns all PodTemplates in a namespace
func (c *ServiceDiscoveryClient) GetPodTemplates(namespace string) ([]*corev1.PodTemplate, error) {
	return c.clientset.CoreV1().PodTemplates(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetReplicationControllers returns all ReplicationControllers in a namespace
func (c *ServiceDiscoveryClient) GetReplicationControllers(namespace string) ([]*corev1.ReplicationController, error) {
	return c.clientset.CoreV1().ReplicationControllers(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetEndpoints returns all Endpoints in a namespace
func (c *ServiceDiscoveryClient) GetEndpoints(namespace string) ([]*corev1.Endpoints, error) {
	return c.clientset.CoreV1().Endpoints(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetEvents returns all Events in a namespace
func (c *ServiceDiscoveryClient) GetEvents(namespace string) ([]*corev1.Event, error) {
	return c.clientset.CoreV1().Events(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetServiceAccounts returns all ServiceAccounts in a namespace
func (c *ServiceDiscoveryClient) GetServiceAccounts(namespace string) ([]*corev1.ServiceAccount, error) {
	return c.clientset.CoreV1().ServiceAccounts(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetRoles returns all Roles in a namespace
func (c *ServiceDiscoveryClient) GetRoles(namespace string) ([]*rbacv1.Role, error) {
	return c.client.RbacV1().Roles(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetClusterRoles returns all ClusterRoles
func (c *ServiceDiscoveryClient) GetClusterRoles() ([]*rbacv1.ClusterRole, error) {
	return c.client.RbacV1().ClusterRoles().List(c.ctx, metav1.ListOptions{})
}

// GetRoleBindings returns all RoleBindings in a namespace
func (c *ServiceDiscoveryClient) GetRoleBindings(namespace string) ([]*rbacv1.RoleBinding, error) {
	return c.client.RbacV1().RoleBindings(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetClusterRoleBindings returns all ClusterRoleBindings
func (c *ServiceDiscoveryClient) GetClusterRoleBindings() ([]*rbacv1.ClusterRoleBinding, error) {
	return c.client.RbacV1().ClusterRoleBindings().List(c.ctx, metav1.ListOptions{})
}

// GetPriorityClasses returns all PriorityClasses
func (c *ServiceDiscoveryClient) GetPriorityClasses() ([]*corev1.PriorityClass, error) {
	return c.clientset.CoreV1().PriorityClasses().List(c.ctx, metav1.ListOptions{})
}

// GetNetworkPolicies returns all NetworkPolicies in a namespace
func (c *ServiceDiscoveryClient) GetNetworkPolicies(namespace string) ([]*networkingv1.NetworkPolicy, error) {
	return c.client.NetworkingV1().NetworkPolicies(namespace).List(c.ctx, metav1.ListOptions{})
}

// GetMutatingWebhookConfigurations returns all MutatingWebhookConfigurations
func (c *ServiceDiscoveryClient) GetMutatingWebhookConfigurations() ([]*rbacv1.MutatingWebhookConfiguration, error) {
	return c.client.AdmissionregistrationV1().MutatingWebhookConfigurations().List(c.ctx, metav1.ListOptions{})
}

// GetValidatingWebhookConfigurations returns all ValidatingWebhookConfigurations
func (c *ServiceDiscoveryClient) GetValidatingWebhookConfigurations() ([]*rbacv1.ValidatingWebhookConfiguration, error) {
	return c.client.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(c.ctx, metav1.ListOptions{})
}
