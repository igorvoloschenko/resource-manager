package api

import (
	"encoding/json"
	"net/http"
	"resource-manager/processing"

	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
)

// getNameSpaceResourceQuota получение назначенной ResourceQuota в namespace
func getNameSpaceResourceQuota(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ns, ok := vars["ns"]
	if !ok {
		http.Error(w, "namespace not specified", http.StatusBadRequest)
		return
	}

	rq, err := processing.GetResourceQuota(ns)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(rq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// getBusinessResourceQuota получение назначенной ResourceQuota у колонны
func getBusinessResourceQuota(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	business, ok := vars["business"]
	if !ok {
		http.Error(w, "business name not specified", http.StatusBadRequest)
		return
	}

	resourcesHard, err := processing.GetResourcesHard(business)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(resourcesHard)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// getBusinessResourceAvailable получение доступных ресурсов у колонны
func getBusinessResourceAvailable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	business, ok := vars["business"]
	if !ok {
		http.Error(w, "business name not specified", http.StatusBadRequest)
		return
	}

	resourceAvailable, err := processing.ResourceAvailable(business)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(resourceAvailable)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// createResourceQuota функция-обработчик по созданию ResourceQuota
func createResourceQuota(w http.ResponseWriter, r *http.Request) {
	body := new(BodyResourceQuota)
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// формирование объекта ResourceQuota с данными из запроса
	newRQ := &corev1.ResourceQuota{ObjectMeta: body.MetaData, Spec: body.Spec}

	// создание DefaultLimitRanges в namespace
	// для задания реквес/лимитов у контейнеров по умолчанию
	if limitrange := r.URL.Query().Get("limitrange"); limitrange != "false" {
		processing.CreateDefaultLimitRanges(body.MetaData.Namespace)
	}

	if err := processing.CreateResourceQuota(newRQ); err != nil {
		if err == processing.ErrNoResourcesAvailable {
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("ok\n"))
}

// updateResourceQuota функция-обработчик по изменению ResourceQuota
func updateResourceQuota(w http.ResponseWriter, r *http.Request) {
	body := new(BodyResourceQuota)
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// формирование объекта ResourceQuota с данными из запроса
	newRQ := &corev1.ResourceQuota{ObjectMeta: body.MetaData, Spec: body.Spec}

	if err := processing.UpdateResourceQuota(newRQ); err != nil {
		if err == processing.ErrNoResourcesAvailable {
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
			return
		}
		if err == processing.ErrRequestedQuotaIsLessUsed {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("ok\n"))

}

// deleteResourceQuota1 удаление ResourceQuota в namespace; данные берутся из url
func deleteResourceQuota1(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ns, ok := vars["ns"]
	if !ok {
		http.Error(w, "namespace not specified", http.StatusBadRequest)
		return
	}

	MetaData := new(BodyResourceQuota).MetaData

	MetaData.Namespace = ns

	rq := &corev1.ResourceQuota{ObjectMeta: MetaData}

	err := processing.DeleteResourceQuota(rq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// deleteResourceQuota2 удаление ResourceQuota в namespace; данные берутся из body
func deleteResourceQuota2(w http.ResponseWriter, r *http.Request) {
	body := new(BodyResourceQuota)
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	MetaData := new(BodyResourceQuota).MetaData

	MetaData.Namespace = body.MetaData.Namespace

	rq := &corev1.ResourceQuota{ObjectMeta: MetaData}

	err = processing.DeleteResourceQuota(rq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// createLimitRange функция-обработчик по созданию LimitRange
func createLimitRange(w http.ResponseWriter, r *http.Request) {
	body := new(BodyLimitRange)
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// формирование объекта LimitRange с данными из запроса
	limitRange := &corev1.LimitRange{ObjectMeta: body.MetaData, Spec: body.Spec}

	err = processing.CreateLimitRanges(limitRange)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("ok\n"))

}

// updateLimitRange функция-обработчик по изменению LimitRange
func updateLimitRange(w http.ResponseWriter, r *http.Request) {
	body := new(BodyLimitRange)
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// формирование объекта LimitRange с данными из запроса
	limitRange := &corev1.LimitRange{ObjectMeta: body.MetaData, Spec: body.Spec}

	err = processing.UpdateLimitRanges(limitRange)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("ok\n"))

}

// deleteLimitRange1 удаление LimitRange в namespace; данные берутся из url
func deleteLimitRange1(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ns, ok := vars["ns"]
	if !ok {
		http.Error(w, "namespace not specified", http.StatusBadRequest)
		return
	}

	MetaData := new(BodyLimitRange).MetaData
	MetaData.Namespace = ns

	limitRange := &corev1.LimitRange{ObjectMeta: MetaData}

	err := processing.DeleteLimitRanges(limitRange)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

// deleteLimitRange2 удаление LimitRange в namespace; данные берутся из body
func deleteLimitRange2(w http.ResponseWriter, r *http.Request) {
	body := new(BodyLimitRange)
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	MetaData := new(BodyLimitRange).MetaData
	MetaData.Namespace = body.MetaData.Namespace

	limitRange := &corev1.LimitRange{ObjectMeta: MetaData}

	err = processing.DeleteLimitRanges(limitRange)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
