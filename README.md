**Новый проект**

Для создания нового проекта необходимо выполнить команду:

**\> tg init someProject**

Где ***someProject*** имя нового проекта.

В процессе создания будет создана базовая структура проекта, транспорт jsonRPC, а так же загружены зависимости.

Описание команды:

**NAME:**
**tg init - init project**

**USAGE:**
**tg init go-msg-project**

**DESCRIPTION:**
**init directory structures, basic configuration package**

**OPTIONS:**
**\--repo value base repository**
**\--jaeger use Jaeger tracer**
**\--zipkin use Zipkin tracer**
**\--mongo enable mongo support**

**Обновление проекта**

В процессе внесения изменений в набор интерфейсов сервиса, перечень их методов, их сигнатур и/или используемых типов,
необходимо актуализировать код транспорта и документацию.

Для это необходимо запустить следующую команду:

**\> tg transport \--services ./pkg/someProject/service ---swagger**

Описание команды:

**NAME:**
**tg transport - generate services transport layer**

**USAGE:**
**tg transport \--main \--jaeger \--swagger**

**OPTIONS:**
**\--services value path to services package**
**\--jaeger use Jaeger tracer**
**\--zipkin use Zipkin tracer (default)**
**\--mongo enable mongo support**
**\--swagger generate swagger docs**

**Документация (swagger)**

Для документирования ***API*** сервиса, его методов и используемых типов данных, можно сгенерировать документацию в
формате ***swagger***.

Генерация ***swagger*** поддерживается для интерфейсов, предоставляющих
***API*** по ***jsonRPC*** и ***HTTP***.

**\> tg swagger**

Описание команды:

**NAME:**
**tg swagger - generate swagger documentation by interfaces**

**USAGE:**
**tg swagger \--iface FirstIface \--iface SecondIface**

**OPTIONS:**
**\--services value path to services package**
**\--iface value interfaces included to swagger**
**\--json save swagger in JSON format**

**Аннотации**

Для управления генератором и другими вспомогательными утилитами, используются аннотации. Аннотации могут иметь пакет,
интерфейс и структуры.

Все аннотации, управляющие генератором **tg**, имеют префикс **@tg**.

Аннотации состоят из переменных. Переменные имеют имя, знак присваивания
(**=**) и значение. Если указано только имя переменной, а знак присваивания и значение опущены, то считается, что
переменная имеет значение равное **True**.

В одной строчке может быть несколько переменных, разделенных пробелами. Если значение переменной включает пробелы, то
его нужно заключить в обратные кавычки «**\`»**.

**Аннотации сервисов**

Для корректной работы генератора ***swagger*** в одном из файлов в пакете ***service*** должны быть размещены следующие
аннотации пакета:

**title** - имя сервиса в документации ***swagger***
**version** - версия документации ***swagger*** сервиса
**description** - описание сервиса в документации ***swagger***
**servers** - список серверов, предоставляющих ***API*** сервиса
**typePrefix** - префикс для типов, используемых в данном сервисе

**Аннотации интерфейсов**

Для управления генерацией кода и документации интерфейса могут применяться следующие аннотации:

**http-server** - генерация ***HTTP*** сервера, предоставляющего ***API*** интерфейса

**jsonRPC-server** - генерация ***jsonRPC*** сервера, предоставляющего ***API*** интерфейса

**metrics** - сбор метрик вызова методов интерфейса
**trace** - трассировка вызова методов интерфейса
**log** - логированное вызова методов интерфейса
**test** - генерация *тестов методом интерфейсов сервиса*

**disableExchange** - запрет генерации типов для вызова методов интерфейса. Используется, если необходимо
кастомизировать данные типы.

**disableEndpoints** - запрет генерации ***endpoint\`ов***. Используется, если необходимо кастомизировать данные типы.

**typePrefix**- префикс для типов, используемых в данном сервисе (имеет приоритет над аннотацией сервиса)

**Аннотации методов**

Для управления генерацией кода и документации методов интерфейса могут применяться следующие аннотации:

**desc** - описание метода в документации ***swagger***. Поддерживает формат ***rich text***.
**summary** - описание метода в документации swagger
**http-encoder** - имя метода для кодирования результата вызова метода, если требуется кастомизация

**http-decoder** - имя метода для декодирования параметров вызова метода, если требуется кастомизация

**http-path** - ***HTTP*** путь в ***URL***, по которому будет доступен метод. В пути допускается указание параметров,
например ***http-path=api/files/{fileId}***.

**http-method** - метод ***HTTP***, который будет соотнесёт с вызовом метода

**http-headers** - список параметров метода, которые будут взяты из
***HTTP*** заголовков. Формат *userID\|x-user-id*, где *userID* - имя параметра метода, *x-user-id* - имя заголовка.
Может содержать список пар, разделённых запятыми.

**http-arg** - список параметров метода, которые будут взяты из аргументов URL. Формат \`*
profileID,count,maxID,sinceID\`*. Разделитель запятая.

**http-request-content-type** - используется для указания списка типов передаваемого контента, отличного от *
application/json* в документации ***swagger***.

**http-response-content-type** - используется для указания списка типов возвращаемого контента, отличного от *
application/json* в документации ***swagger***. Разделитель вертикальная черта «\|»

**log-skip** - пропуск полей при логировании, имена полей указываются через запятую «,»

**disable-http** - указание генератору пропустить создание ***HTTP*** реализации данного метода

**disable-jsonRPC** - указание генератору пропустить создание ***jsonRPC*** реализации данного метода

**Аннотации типов**

Для управления генерацией документации типов, используемых в методах интерфейсов могут применяться следующие аннотации:

**type** - переопределение типа в результирующем swagger
**example** - пример значения. Может иметь как простой тип, так представлять собой json-объект.