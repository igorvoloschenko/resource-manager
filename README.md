# Resource-manager
Сервис реализует назначение квот на неймспейсы по доступным ресурсам у колонны и работает в виде RestAPI предоставляя следующие endpoints:
- /v1/resourcequotas
- /v1/limitranges (как дополнительная возможность)

Добавлены следующие endpoints которые поддерживают методы GET:
- /v1/namespace/<имя namespace>/resourcequotas - установленные квоты в namespace

- /v1/business/<имя бизнес колонны>/resourcequotas - установленные квоты у колонны

- /v1/business/<имя бизнес колонны>/resourceavailable - доступные ресурсы у колонны для установки квот


#### Язык программирования: 
 - Go

#### Бэкенд у сервиса:
- kubernetes/openshift
- prometheus

#### Зависимости сервиса
Сервис для задание квот использует данные из prometheus и kubernetes/openshift.

Prometheus:
- получение закупленных ресурсов у колонны: метрики cap_asset_cpu_total и cap_asset_memory_bytes_total
- получение зарезервированных ресурсов в кластере у колонны: метрики cap_quote_hard_cpu_total и cap_quote_hard_memory_bytes_total

Kubernetes/Openshift:
- берутся данные имени колонны по имени namespace
- получение текущих используемых ресурсов у namespace

Метрики cap_asset_cpu_total, cap_asset_memory_bytes_total формируются экспортером snipeit-exporter. Данные для этих метрик берутся из сервиса snipe-it.

#### Использование сервиса
Типы запросов:

Задание квот для namespace:
```
POST /v1/resourcequotas
{
    "metadata": {
        "namespace": "platform"
    },
    "spec": {
        "hard": {
            "limits.cpu": "8",
            "limits.memory": "32Gi"
        }
    }
}
```

При создании квоты - задается limitrange по умолчанию. Данные по созданию limitrange берутся из конфигурации сервиса processing.default_limitrange.

Параметр `limitrange=false` отключает создание limitrange

Пример:
```
POST /v1/resourcequotas?limitrange=false
{
    "metadata": {
        "namespace": "platform"
    },
    "spec": {
        "hard": {
            "limits.cpu": "8",
            "limits.memory": "32Gi"
        }
    }
}
```

Изменение квот для namespace:
```
PUT /v1/resourcequotas
{
    "metadata": {
        "namespace": "platform"
    },
    "spec": {
        "hard": {
            "limits.cpu": "1",
            "limits.memory": "1Gi"
        }
    }
}
```

При выставлении квоты сервис проверяет наличие запрошенных ресурсов и текущих. При нехватке ресурсов сервис не задает/изменяет квоту и сообщает об этом http-кодом 412 и сообщением в ответе. Если запрашиваемая квота меньше used, квота не изменяется и сервис отвечает кодом 409.

http коды ответов:
- 200: запрос выполнен успешно
- 400: неверный запрос (когда передаются некорректные данные в запросе)
- 409: конфликт (если запрашиваемая квота меньше текущего значения quota resource used у неймспейса)
- 412: предварительное условие не выполнено (недостаточно ресурсов)
- 500: внутренняя ошибка при обработке запроса(это может быть недоступность prometheus, ошибка на стороне кластера openshift/kubernetes или другого рода внутренних ошибок)

Дополнительно добавлена возможность для создания/изменения limitrange в namespace.

Создание limitrange:

```
POST /v1/limitranges
{
    "metadata": {
        ...
    },
    "spec": {
        ...
    }
}
```

Изменение limitrange:

```
PUT /v1/limitranges
{
    "metadata": {
        ...
    },
    "spec": {
        ...
    }
}
```

#### Сборка и настройка сервиса

В проекте для сборки docker-образа используется скрипт `build.sh`
Скрипт может принимать аргументы:
```
./build.sh <NameImage>
```
При запуске скрипта без аргументов формируется образ с именем resource-manager

Конфигурация сервиса по умолчанию берется из файла config.yaml в рабочей директории или путь к конфигурации указывается в переменной окружения APP_CONFIG_PATH=<path>/<namefileyaml>, пример: export APP_CONFIG_PATH=/app/config.yaml

Пример конфигурации:

```
# порт и адрес интерфейса на котором будет работать сервис
listen_addr: ":8080"

processing:
  # имя ResourceQuota по умолчанию
  default_resource_quota_name: cap-resource

  # процент удержания ресурсов на внутренние сервисы контура
  infra_fee: 15
  # список колонн для инфраструктурных сервисов
  infra_customers: 
  - kc
  - infrastructure

  # поле в аннотации к namespace по которому определяется имя бизнес-коллоны
  business_annotation_field_name: business.unit

  # значение по умолчанию для создания LimitRange в namespace
  default_limitrange:
    metadata:
      name: cap-limitrange
    spec:
      limits:
      - default:
          memory: 128Mi
          cpu: 100m
        defaultRequest:
          memory: 128Mi
          cpu: 100m
        type: "Container"

# блок для подключения к prometheus
prometheus:
  addr: http://localhost:9090/
```

#### Разворачивание сервиса в кластере
В проекте есть директория helm для разворачивания сервиса в кластере kubernetes/openshift с помощью утилиты helm.