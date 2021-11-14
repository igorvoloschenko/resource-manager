package api

import (
	"net/http"
	"resource-manager/config"
	"resource-manager/kube"
	"resource-manager/processing"
	"resource-manager/prometheus"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"
	log "k8s.io/klog/v2"
)

func logRequestHendler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(h, w, r)
		log.Infoln(r.RemoteAddr, r.Method, r.URL, m.Code, m.Duration, m.Written)
	})
}

func Run(cfg *config.Config) error {

	router := mux.NewRouter().StrictSlash(true)

	// указание обработчиков для endpoints
	router.HandleFunc(
		"/v1/namespace/{ns}/resourcequotas", getNameSpaceResourceQuota).Methods("GET")

	router.HandleFunc(
		"/v1/business/{business}/resourcequotas", getBusinessResourceQuota).Methods("GET")

	router.HandleFunc(
		"/v1/business/{business}/resourceavailable", getBusinessResourceAvailable).Methods("GET")

	router.HandleFunc("/v1/resourcequotas", createResourceQuota).Methods("POST")
	router.HandleFunc("/v1/resourcequotas", updateResourceQuota).Methods("PUT")
	// router.HandleFunc("/v1/resourcequotas", deleteResourceQuota1).Methods("DELETE")
	// router.HandleFunc("/v1/namespace/{ns}/resourcequotas", deleteResourceQuota2).Methods("DELETE")
	router.HandleFunc("/v1/limitranges", createLimitRange).Methods("POST")
	router.HandleFunc("/v1/limitranges", updateLimitRange).Methods("PUT")
	// router.HandleFunc("/v1/namespace/{ns}/deleteLimitRange", deleteLimitRange1).Methods("DELETE")
	// router.HandleFunc("/v1/limitranges", deleteLimitRange2).Methods("DELETE")

	// инициализация клиента в пакете kube для работы с kubernetes
	err := kube.Init()
	if err != nil {
		return err
	}

	// инициализация клиента в пакете prometheus для работы с prometheus
	err = prometheus.Init(cfg.Prometheus)
	if err != nil {
		return err
	}

	// инициализация конфигурации в пакете processing
	err = processing.Init(cfg.Processing)
	if err != nil {
		return err
	}

	// запуск сервера API
	addr := cfg.Addr
	if addr == "" {
		addr = ":8080"
	}
	log.Infof("Service start, listen and serve: \"%s\"", addr)
	return http.ListenAndServe(addr, logRequestHendler(router))
}
