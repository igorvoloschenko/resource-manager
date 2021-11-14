package prometheus

import (
	"context"
	"resource-manager/config"
	"time"

	log "k8s.io/klog/v2"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var client api.Client

func Init(cfg config.PrometheusType) error {
	var err error
	client, err = api.NewClient(api.Config{
		Address: cfg.Address,
	})
	if err != nil {
		return err
	}
	return nil
}

func getVector(req string) (model.Vector, error) {
	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := v1api.Query(ctx, req, time.Now())
	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		log.Warning("Prometheus: %v\n", warnings)
	}

	return result.(model.Vector), nil
}

func GetValue(req string) (float64, error) {
	vector, err := getVector(req)
	if err != nil {
		return 0, err
	}

	if len(vector) == 0 {
		log.Warningf("Prometheus: no data on request %s", req)
		return 0, nil
	}

	return float64(vector[0].Value), nil
}
