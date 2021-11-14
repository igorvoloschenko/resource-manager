package processing

import "errors"

var (
	// ErrNoResourcesAvailable нет доступных ресурсов
	ErrNoResourcesAvailable = errors.New("no resources available")
	// ErrRequestedQuotaIsLessUsed запрошенная квота меньше используемых ресурсов
	ErrRequestedQuotaIsLessUsed = errors.New("requested quota is less resources used")
)
