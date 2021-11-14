package processing

import (
	"fmt"
	"resource-manager/config"
	"resource-manager/kube"
	"resource-manager/prometheus"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	log "k8s.io/klog/v2"
)

const (
	BUSINESS_FIELD_NAME         = "business.unit"
	DEFAULT_RESOURCE_QUOTA_NAME = "cap-resource"
	DEFAULT_INFRA_FEE           = 15
	DEFAULT_CPUOVERSUBSCRIPTION = 1.7
)

type CalculateType struct {
	InfraFee            int
	InfraCustomers      []string
	CpuOversubscription float64
}

type ConfigProcessing struct {
	DefaultResourceQuotaName    string
	Calculate                   CalculateType
	BusinessAnnotationFieldName string
	DefaultLimitRange           *corev1.LimitRange
}

var cfg ConfigProcessing

// Init инициализация конфигурации
func Init(c config.ProcessingType) error {
	// convert string InfraFee from config to float64
	if c.InfraFee != "" {
		infraFee, err := strconv.Atoi(c.InfraFee)
		if err != nil {
			return err
		}
		cfg.Calculate.InfraFee = infraFee
	} else {
		cfg.Calculate.InfraFee = DEFAULT_INFRA_FEE
	}

	// convert string CpuOversubscription from config to float64
	if c.CpuOversubscription != "" {
		CpuOversubscription, err := strconv.ParseFloat(c.CpuOversubscription, 64)
		if err != nil {
			return err
		}
		cfg.Calculate.CpuOversubscription = CpuOversubscription
	} else {
		cfg.Calculate.CpuOversubscription = float64(DEFAULT_CPUOVERSUBSCRIPTION)
	}

	cfg.Calculate.InfraCustomers = c.InfraCustomers

	json := jsoniter.ConfigCompatibleWithStandardLibrary

	// marshal to json object map[string]interface{}
	b, err := json.Marshal(c.DefaultLimitRange)
	if err != nil {
		return err
	}

	cfg.DefaultLimitRange = new(corev1.LimitRange)
	err = json.Unmarshal(b, cfg.DefaultLimitRange)
	if err != nil {
		return err
	}

	cfg.DefaultResourceQuotaName = c.DefaultResourceQuotaName
	if cfg.DefaultResourceQuotaName == "" {
		cfg.DefaultResourceQuotaName = DEFAULT_RESOURCE_QUOTA_NAME
	}

	cfg.BusinessAnnotationFieldName = c.BusinessAnnotationFieldName
	if cfg.BusinessAnnotationFieldName == "" {
		cfg.BusinessAnnotationFieldName = BUSINESS_FIELD_NAME
	}

	return nil
}

// stringInSlice проверка на наличие строки в slice(в списке из строк)
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// infoResourceList формирование строки для логирования
// <nameResource1>: <valueResource1>, <nameResource2>: <valueResource2>, ...
func infoResourceList(rl corev1.ResourceList) string {
	strResource := []string{}
	q := resource.Quantity{}
	for rname := range rl {
		q = rl[rname]
		strResource = append(strResource, fmt.Sprintf("%v: %v", rname, q.String()))
	}
	return strings.Join(strResource, ", ")
}

// infoResourceQuota формирование строки для логирования с данными квоты
// {name: <имя квоты>, namespace: <имя namespace>,
//  <nameResource1>: <valueResource1>, <nameResource2>: <valueResource2>, ...}
func infoResourceQuota(rq *corev1.ResourceQuota) string {
	return fmt.Sprintf(
		"{name: %s, namespace: %s, %s}",
		rq.Name,
		rq.Namespace,
		infoResourceList(rq.Spec.Hard),
	)
}

// getResourceFromProm получение ресурсов с prometheus и формирование corev1.ResourceList
func getResourceFromProm(queries map[corev1.ResourceName]string) (corev1.ResourceList, error) {
	rl := make(corev1.ResourceList)
	q := resource.Quantity{}

	for rname := range queries {
		v, err := prometheus.GetValue(queries[rname])
		if err != nil {
			return nil, err
		}
		q.Set(int64(v))
		rl[rname] = q.DeepCopy()
	}

	return rl, nil
}

// GetResourcesAsset получение закупленных ресурсов у колонны
func GetResourcesAsset(business string) (corev1.ResourceList, error) {
	return getResourceFromProm(map[corev1.ResourceName]string{
		corev1.ResourceLimitsCPU: fmt.Sprintf(
			"sum(cap_asset_cpu_total{customer=~\"(?i:%s)\"})", business,
		),
		corev1.ResourceLimitsMemory: fmt.Sprintf(
			"sum(cap_asset_memory_bytes_total{customer=~\"(?i:%s)\"})", business,
		),
	})
}

// SumResources сумма ресурсов rl1 + rl2
func SumResources(rl1, rl2 corev1.ResourceList) corev1.ResourceList {
	for rname, q := range rl1 {
		q.Add(rl2[rname])
		rl2[rname] = q

	}
	return rl2
}

// SubResources возвращает разницу в ресурсах между rl1 и rl2
func SubResources(rl1, rl2 corev1.ResourceList) corev1.ResourceList {
	for rname, q := range rl1 {
		q.Sub(rl2[rname])
		rl2[rname] = q

	}
	return rl2
}

// GetResourcesHard получение установленных квот на ресурсы в кластере у колонны
func GetResourcesHard(business string) (corev1.ResourceList, error) {
	return getResourceFromProm(map[corev1.ResourceName]string{
		corev1.ResourceLimitsCPU: fmt.Sprintf(
			"sum(cap_quote_hard_cpu_total{customer=~\"(?i:%s)\"})", business,
		),
		corev1.ResourceLimitsMemory: fmt.Sprintf(
			"sum(cap_quote_hard_memory_bytes_total{customer=~\"(?i:%s)\"})", business,
		),
	})
}

// ResourceAvailable получение доступных ресурсов у колонны
func ResourceAvailable(business string) (corev1.ResourceList, error) {
	var (
		err error
	)

	resourcesAsset := corev1.ResourceList{}
	resourcesHard := corev1.ResourceList{}

	// если колонна входит в список infra_customers,
	// то суммируются все ресурсы закупленные и запрошенные по колоннам из infra_customers
	if stringInSlice(business, cfg.Calculate.InfraCustomers) {
		for _, customer := range cfg.Calculate.InfraCustomers {

			// Получение закупленных ресурсов у колонны
			resourcesAssetCustomer, err := GetResourcesAsset(customer)
			if err != nil {
				log.Errorf("get resources asset: %s", err)
				return nil, err
			}

			// Получение запрошенных ресурсов в кластере у колонны
			resourcesHardCustomer, err := GetResourcesHard(customer)
			if err != nil {
				log.Errorf("get resources hard: %s", err)
				return nil, err
			}

			log.Infof(
				"resourcesHardCustomer on the customer %s: {%s}",
				customer,
				infoResourceList(resourcesHardCustomer),
			)

			// суммирование ресурсов
			resourcesAsset = SumResources(resourcesAsset, resourcesAssetCustomer)

			resourcesHard = SumResources(resourcesHard, resourcesHardCustomer)
		}

		// расчет для колонн не входящих в infra_customers
	} else {
		resourcesAsset, err = GetResourcesAsset(business)
		if err != nil {
			log.Errorf("get resources total: %s", err)
			return nil, err
		}

		// Получение запрошенных ресурсов в кластере у колонны
		resourcesHard, err = GetResourcesHard(business)
		if err != nil {
			log.Errorf("get resources hard: %s", err)
			return nil, err
		}
	}

	log.Infof(
		"resourcesAsset on the business %s: {%s}",
		business,
		infoResourceList(resourcesAsset),
	)

	// Получение ресурсов с учетом данных из cfg.Calculate
	resourcesCalculate, err := CalculateResources(resourcesAsset, business, cfg.Calculate)
	if err != nil {
		log.Errorf("calculate resources: %s", err)
		return nil, err
	}

	log.Infof(
		"resourcesCalculate on the business %s: {%s}",
		business,
		infoResourceList(resourcesCalculate),
	)

	log.Infof(
		"resourcesHard on the business %s: {%s}",
		business,
		infoResourceList(resourcesHard),
	)

	// Получение разницы между resourcesCalculate и resourcesHard
	resourcesDiff := SubResources(resourcesCalculate, resourcesHard)

	return resourcesDiff, nil

}

// IsResourcesAvailable доступны ли ресурсы у колонны
func IsResourcesAvailable(business string, rl corev1.ResourceList) (bool, error) {

	resourceAvailable, err := ResourceAvailable(business)
	if err != nil {
		log.Errorf("get resources available: %s", err)
		return false, err
	}
	return geResource(resourceAvailable, rl), nil
}

// CalculateResources расчет доступных ресурсов на основе закупленных и данных в calculateCfg
func CalculateResources(rl corev1.ResourceList, business string, calculateCfg CalculateType) (corev1.ResourceList, error) {
	var (
		q     resource.Quantity
		value float64
	)
	resourcesAvailable := corev1.ResourceList{}

	// проверяем входит ли имя колонны в список infra_customers
	if stringInSlice(business, calculateCfg.InfraCustomers) {
		// если колонна входит в этот список
		// то производится расчет ресурсов с учётом закупленных для этой колонны
		// и процента infra_fee от остальных колонн

		infraCustomers := strings.Join(calculateCfg.InfraCustomers, "|")

		promReqMap := map[corev1.ResourceName]string{
			corev1.ResourceLimitsCPU: fmt.Sprintf(
				"sum(cap_asset_cpu_total{customer!~\"(?i:%s)\"})", infraCustomers),
			corev1.ResourceLimitsMemory: fmt.Sprintf(
				"sum(cap_asset_memory_bytes_total{customer!~\"(?i:%s)\"})", infraCustomers),
		}

		for rname := range rl {
			q = rl[rname]
			promValue, err := prometheus.GetValue(promReqMap[rname])
			if err != nil {
				return rl, err
			}

			value = float64(q.Value()) + (promValue * float64(calculateCfg.InfraFee) / 100)
			if rname == corev1.ResourceLimitsCPU {
				value = value * calculateCfg.CpuOversubscription
			}
			q.Set(int64(value))
			resourcesAvailable[rname] = q.DeepCopy()
		}

	} else {
		// для остальных колонн удерживается процент infra_fee от закупленных ресурсов
		for rname := range rl {
			q = rl[rname]
			value = float64(q.Value()) - (float64(q.Value()) * float64(calculateCfg.InfraFee) / 100)

			// для ресурса ResourceLimitsCPU учитывается коэффициент передописки
			if rname == corev1.ResourceLimitsCPU {
				value = value * calculateCfg.CpuOversubscription
			}

			q.Set(int64(value))
			resourcesAvailable[rname] = q.DeepCopy()
		}
	}
	return resourcesAvailable, nil
}

// geResource больше или равно rl1 >= rl2
func geResource(rl1, rl2 corev1.ResourceList) bool {
	var q1, q2 resource.Quantity
	for rname := range rl1 {
		q1 = rl1[rname]
		q2 = rl2[rname]
		if !(q1.Value() >= q2.Value()) {
			return false
		}
	}
	return true
}

// CreateResourceQuota создание квоты на ресурсы
func CreateResourceQuota(rq *corev1.ResourceQuota) error {
	if rq.Name == "" {
		rq.Name = cfg.DefaultResourceQuotaName
	}

	namespace, err := kube.GetNamespace(rq.Namespace)
	if err != nil {
		log.Errorf("Get namespace: %s", err)
		return err
	}

	businessName, err := GetBusinessName(namespace)
	if err != nil {
		log.Errorf("Get business name: %s", err)
		return err
	}

	if ok, err := IsResourcesAvailable(businessName, rq.Spec.Hard); !ok {
		if err != nil {
			return err
		}
		return ErrNoResourcesAvailable
	}

	_, err = kube.CreateQuota(rq)
	if err != nil {
		log.Errorf("Create resource quota: %s; %v", infoResourceQuota(rq), err)
		return err
	}
	log.Infof("Create resource quota: %s; OK", infoResourceQuota(rq))
	return nil
}

// UpdateResourceQuota обновление квоты на ресурсы
func UpdateResourceQuota(rq *corev1.ResourceQuota) error {
	if rq.Name == "" {
		rq.Name = cfg.DefaultResourceQuotaName
	}

	currentRQ, err := GetResourceQuota(rq.Namespace)
	if err != nil {
		return err
	}

	// является ли запрашиваемая квота больше или равна used текущей квоты
	// иначе выход с ошибкой ErrRequestedQuotaIsLessUsed
	resourcesDiff := make(corev1.ResourceList)
	if !geResource(rq.Spec.Hard, currentRQ.Status.Used) {
		return ErrRequestedQuotaIsLessUsed
	}

	namespace, err := kube.GetNamespace(rq.Namespace)
	if err != nil {
		log.Errorf("Get namespace: %s", err)
		return err
	}

	businessName, err := GetBusinessName(namespace)
	if err != nil {
		log.Errorf("Get business name: %s", err)
		return err
	}

	resourcesDiff = SubResources(rq.Spec.Hard, currentRQ.Spec.Hard)

	if ok, err := IsResourcesAvailable(businessName, resourcesDiff); !ok {
		if err != nil {
			return err
		}
		return ErrNoResourcesAvailable
	}

	_, err = kube.UpdateQuota(rq)
	if err != nil {
		log.Errorf("Update resource quota: %s; %v", infoResourceQuota(rq), err)
		return err
	}
	log.Infof("Update resource quota: %s; OK", infoResourceQuota(rq))
	return nil
}

// DeleteResourceQuota удаление квоты на ресурсы
func DeleteResourceQuota(rq *corev1.ResourceQuota) error {
	if rq.Name == "" {
		rq.Name = cfg.DefaultResourceQuotaName
	}
	err := kube.DeleteQuota(rq)
	if err != nil {
		log.Errorf("Delete resource quota: %s; %v", infoResourceQuota(rq), err)
		return err
	}
	log.Infof("Delete resource quota: %s; OK", infoResourceQuota(rq))
	return nil
}

// GetBusinessName получение имени бизнесс колонны по имени неймспейса
// имя переводится в нижний регистр
func GetBusinessName(namespace *corev1.Namespace) (string, error) {
	// namespace, err := kube.GetNamespace(ns)
	// if err != nil {
	// 	return "", err
	// }
	annotaions := namespace.Annotations
	if len(annotaions) == 0 {
		return "", fmt.Errorf("annotation in namespace %s is empty", namespace.Name)
	}

	businessName, ok := annotaions[cfg.BusinessAnnotationFieldName]
	if !ok {
		return "", fmt.Errorf(
			"in the annotation, the %s field in the namespace %s is not filled",
			cfg.BusinessAnnotationFieldName,
			namespace.Name,
		)
	}

	return strings.ToLower(businessName), nil
}

// GetResourceQuota получение текущей resourcequota по имени неймспейса
func GetResourceQuota(ns string) (*corev1.ResourceQuota, error) {
	return kube.GetQuota(cfg.DefaultResourceQuotaName, ns)
}

// CreateLimitRanges создание LimitRange
func CreateLimitRanges(lr *corev1.LimitRange) error {
	_, err := kube.CreateLimitRanges(lr)
	if err != nil {
		log.Errorf("Create limit ranges: %v", err)
	}
	return err
}

// CreateDefaultLimitRanges создание LimitRange со значением по умолчанию
func CreateDefaultLimitRanges(ns string) error {
	limitRange := *cfg.DefaultLimitRange
	limitRange.Namespace = ns
	return CreateLimitRanges(&limitRange)
}

// UpdateLimitRanges обновление LimitRange
func UpdateLimitRanges(lr *corev1.LimitRange) error {
	_, err := kube.UpdateLimitRanges(lr)
	if err != nil {
		log.Errorf("Update limit ranges: %v", err)
	}
	return err
}

// DeleteLimitRanges удаление LimitRange
func DeleteLimitRanges(lr *corev1.LimitRange) error {
	if lr.GetName() == "" {
		lr.Name = cfg.DefaultLimitRange.Name
	}
	err := kube.DeleteLimitRanges(lr)
	if err != nil {
		log.Errorf("Delete limit ranges: %v", err)
	}
	return err
}
