Текстовое описание анализа работы приложения.

Профили снимались после нагрузки: K6 — 4,25 миллиона запросов за 9,5 минут (примерно 7500 запросов в секунду).

Анализ памяти (inuse_space) показал, что она используется в небольшом количестве и утечек нет: значение возвращается примерно к начальному через некоторое время после нагрузки. 

Поэтому первым направлением для оптимизации выбраны высокие показатели аллокаций: alloc_objects — ~402 млн, alloc_space — ~30,59 ГБ.

Под нагрузкой был выявлен дедлок на обновление метрик в базе: запросы стартовали параллельно, а список метрик был не отсортирован — исправлено.

GetNamesList При выводе всех метрик pgx.CollectRows заменён на цикл for rows.Next(); процент от alloc_objects перешёл в alloc_space.
Предвыделение памяти для списка метрик (256) снизило аллокации на пару процентов.

ListMetrics Переход с fmt.Sprintf + strings.Join на html/template оказался неудачным — стало хуже. Решил применять html/template используется более активно, применять его для одного метода нецелесообразно — убрал.
ListMetrics Метод вывода списка переведён на strings.Builder с предвыделением буфера — это показало снижение аллокаций.
Далее был установлен кэш на этот метод — скорее не как оптимизация кода, а чтобы понять, вылетит ли он полностью из отчёта. С первых мест отчёта данный метод ушёл; кэш оставлен.

Введён recorderPool (sync.Pool) для middleware Logger — аллокации объектов снизились.

Заменены написанные мной middleware сжатия на сжатие, поддерживаемое фреймворком chi, — количество аллокаций ещё уменьшилось.

В SignResponseMiddleware тоже введён sync.Pool — с положительным результатом.

Результаты
По итогу общее количество аллокаций снизилось примерно на 30%.

Немного увеличилось использование памяти (inuse_space) — примерно на 5 МБ. Это связано с кэшем и sync.Pool, которые теперь находятся в памяти.

Лучшей оптимизацией стало не использование собственного велосипеда при сжатии. p95 даже немного вырос(плохо и не понятно), RPS  остался на начальном уровне

Benchmark

korob\internal\repository\dbpg.go Get
Стартовые значения
BenchmarkPgStorageGet/ExistingMetric-32                 4935        229087 ns/op         751 B/op         14 allocs/op
BenchmarkPgStorageGet/NonExistingMetric-32              5220        234977 ns/op         701 B/op         13 allocs/op

Подрубил config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
BenchmarkPgStorageGet/ExistingMetric-32                 4436        232255 ns/op         751 B/op         14 allocs/op
BenchmarkPgStorageGet/NonExistingMetric-32              5265        226469 ns/op         702 B/op         13 allocs/op

Есть небольшой положительный эффект от включения кэширования операторов

korob\internal\repository\dbpg.go Save
Стартовые значения
BenchmarkPgStorageSave/InsertNewCounter-32              4876        252480 ns/op         243 B/op          8 allocs/op
BenchmarkPgStorageSave/InsertNewGauge-32                3788        274513 ns/op         243 B/op          8 allocs/op
BenchmarkPgStorageSave/UpdateExistingCounter-32         4405        271340 ns/op         225 B/op          7 allocs/op
BenchmarkPgStorageSave/UpdateExistingGauge-32           4449        267442 ns/op         226 B/op          7 allocs/op

Просчет значение перенесен в запрос
BenchmarkPgStorageSave/InsertNewCounter-32         	    4173	    265152 ns/op	     244 B/op	       8 allocs/op
BenchmarkPgStorageSave/InsertNewGauge-32           	    4420	    260272 ns/op	     242 B/op	       8 allocs/op
BenchmarkPgStorageSave/UpdateExistingCounter-32    	    4665	    264450 ns/op	     226 B/op	       7 allocs/op
BenchmarkPgStorageSave/UpdateExistingGauge-32      	    4808	    277856 ns/op	     225 B/op	       7 allocs/op

Изменения в рамках погрешностей, можем сделать вывод что изменение логики работы не ухудшило производительность

korob\internal\repository\dbpg.go BatchSave
Стартовые значения
BenchmarkPgStorageBatchSave/BatchSize10_Insert-32               2872        423589 ns/op        3450 B/op         69 allocs/op
BenchmarkPgStorageBatchSave/BatchSize100_Insert-32               738       1679272 ns/op       76435 B/op        777 allocs/op
BenchmarkPgStorageBatchSave/BatchSize10_Update-32               2935        374913 ns/op        3360 B/op         59 allocs/op

Оптимизации сборки запроса
BenchmarkPgStorageBatchSave/BatchSize10_Insert-32         	    2834	    411117 ns/op	    2508 B/op	      64 allocs/op
BenchmarkPgStorageBatchSave/BatchSize100_Insert-32        	     769	   1747211 ns/op	   69701 B/op	     623 allocs/op
BenchmarkPgStorageBatchSave/BatchSize10_Update-32         	    2674	    417041 ns/op	    2423 B/op	      55 allocs/op
Есть улучшения по памяти и уменьшились алокации


go tool pprof -top -diff_base=profiles/base.pprof -alloc_objects profiles/result.pprof
File: main
Build ID: 4fc71448bf5b904075b7d7b9df3fe4a1e28fdc5e
Type: alloc_objects
Time: 2026-10-11 19:16:07 MSK
Showing nodes accounting for -118910494, 29.58% of 402041097 total
Dropped 346 nodes (cum <= 2010205)
      flat  flat%   sum%        cum   cum%
 -81396953 20.25% 20.25% -117322187 29.18%  github.com/mersikovs/korob.git/internal/repository.(*PgStorage).GetNamesList.func1
 -35663088  8.87% 29.12%  -35663088  8.87%  github.com/jackc/pgx/v5/pgtype.scanPlanString.Scan
  -7853756  1.95% 31.07% -127002659 31.59%  github.com/mersikovs/korob.git/internal/router.NewRouter.SignResponseMiddleware.func3.1
   4571209  1.14% 29.93%    4571209  1.14%  strings.genSplit
   4554821  1.13% 28.80%    4554821  1.13%  errors.New (inline)
   4319151  1.07% 27.73%  192543096 47.89%  github.com/go-chi/chi/v5/middleware.(*Compressor).Handler-fm.(*Compressor).Handler.func1
  -4229447  1.05% 28.78% -118725478 29.53%  github.com/mersikovs/korob.git/internal/router.NewRouter.Logger.func1.1
  -1494647  0.37% 29.15% -119111750 29.63%  github.com/jackc/pgx/v5.AppendRows[go.shape.string,go.shape.[]go.shape.string]
    950287  0.24% 28.91%     950287  0.24%  reflect.unsafe_New
   -890737  0.22% 29.13%    -890737  0.22%  net/http.Header.Clone (inline)
   -786480   0.2% 29.33%    -989230  0.25%  bytes.(*Buffer).grow
   -753675  0.19% 29.52%    -753675  0.19%  net/textproto.canonicalMIMEHeaderKey
   -748219  0.19% 29.70%    -740027  0.18%  net/textproto.(*Reader).ReadLine (inline)
    655380  0.16% 29.54%     675383  0.17%  github.com/mersikovs/korob.git/internal/audit.(*Dispatcher).Publish
    507918  0.13% 29.41%     774349  0.19%  github.com/mersikovs/korob.git/internal/handler.(*MetricHandler).UpdateMetricFromBody
    458759  0.11% 29.30% -117932344 29.33%  net/http.(*conn).serve
   -434174  0.11% 29.41%  -36097262  8.98%  github.com/jackc/pgx/v5.(*baseRows).Scan
   -425978  0.11% 29.51%    1778128  0.44%  github.com/mersikovs/korob.git/internal/repository.(*PgStorage).Get
   -277027 0.069% 29.58% -121600027 30.25%  github.com/mersikovs/korob.git/internal/handler.(*MetricHandler).ListMetrics
   -102032 0.025% 29.61%     770353  0.19%  github.com/mersikovs/korob.git/internal/handler.(*MetricHandler).BatchUpdateMetricsFromBody
     62278 0.015% 29.59%    -873319  0.22%  net/http.readRequest
     33309 0.0083% 29.58% -121306283 30.17%  github.com/mersikovs/korob.git/internal/repository.(*PgStorage).GetNamesList
     32768 0.0082% 29.58%    1586222  0.39%  github.com/mersikovs/korob.git/internal/service.(*MetricService).GetMetric
      -161 4e-05% 29.58%     589908  0.15%  github.com/jackc/pgx/v5.(*ExtendedQueryBuilder).appendParam
         0     0% 29.58%    -989230  0.25%  bytes.(*Buffer).Write
         0     0% 29.58%     954750  0.24%  encoding/json.(*decodeState).array
         0     0% 29.58%     670753  0.17%  encoding/json.(*decodeState).unmarshal
         0     0% 29.58%     670753  0.17%  encoding/json.(*decodeState).value
         0     0% 29.58% -118398646 29.45%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 29.58% -119202432 29.65%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 29.58%    4292673  1.07%  github.com/go-chi/chi/v5/middleware.(*Compressor).selectEncoder
         0     0% 29.58%    4554821  1.13%  github.com/go-chi/chi/v5/middleware.(*compressResponseWriter).Close
         0     0% 29.58%   12265319  3.05%  github.com/go-chi/chi/v5/middleware.(*compressResponseWriter).WriteHeader
         0     0% 29.58%    1089645  0.27%  github.com/jackc/pgx/v5.(*Conn).Exec
         0     0% 29.58%    1660561  0.41%  github.com/jackc/pgx/v5.(*Conn).QueryRow (inline)
         0     0% 29.58%    1089645  0.27%  github.com/jackc/pgx/v5.(*Conn).exec
         0     0% 29.58%    1099202  0.27%  github.com/jackc/pgx/v5.(*Conn).execPrepared
         0     0% 29.58%     589908  0.15%  github.com/jackc/pgx/v5.(*ExtendedQueryBuilder).Build
         0     0% 29.58%     590069  0.15%  github.com/jackc/pgx/v5.(*ExtendedQueryBuilder).encodeExtendedParamValue
         0     0% 29.58% -119111750 29.63%  github.com/jackc/pgx/v5.CollectRows[go.shape.string] (inline)
         0     0% 29.58%     590069  0.15%  github.com/jackc/pgx/v5/pgtype.(*Map).Encode
         0     0% 29.58%     588809  0.15%  github.com/jackc/pgx/v5/pgtype.(*derefPointerEncodePlan).Encode
         0     0% 29.58%    1089645  0.27%  github.com/jackc/pgx/v5/pgxpool.(*Conn).Exec
         0     0% 29.58%   -1419593  0.35%  github.com/jackc/pgx/v5/pgxpool.(*Conn).Query
         0     0% 29.58%    1660561  0.41%  github.com/jackc/pgx/v5/pgxpool.(*Conn).QueryRow
         0     0% 29.58%    1192329   0.3%  github.com/jackc/pgx/v5/pgxpool.(*Pool).Exec
         0     0% 29.58%   -2293379  0.57%  github.com/jackc/pgx/v5/pgxpool.(*Pool).Query
         0     0% 29.58%    2441671  0.61%  github.com/jackc/pgx/v5/pgxpool.(*Pool).QueryRow
         0     0% 29.58%  -35859697  8.92%  github.com/jackc/pgx/v5/pgxpool.(*poolRows).Scan
         0     0% 29.58%    -989230  0.25%  github.com/mersikovs/korob.git/internal/middleware.(*responseCapture).Write
         0     0% 29.58%    -904640  0.23%  github.com/mersikovs/korob.git/internal/middleware.(*responseRecorder).WriteHeader
         0     0% 29.58% -307141685 76.40%  github.com/mersikovs/korob.git/internal/middleware.GzipRequestMiddleware.func1
         0     0% 29.58% -307142164 76.40%  github.com/mersikovs/korob.git/internal/middleware.GzipResponseMiddleware.func1
         0     0% 29.58% -127764721 31.78%  github.com/mersikovs/korob.git/internal/router.NewRouter.VerifySha256Middleware.func2.1
         0     0% 29.58% -121306283 30.17%  github.com/mersikovs/korob.git/internal/service.(*MetricService).ListMetrics
         0     0% 29.58%     703188  0.17%  github.com/mersikovs/korob.git/internal/signing.CalcHash
         0     0% 29.58%    -655369  0.16%  net/http.(*body).Read
         0     0% 29.58%    -655369  0.16%  net/http.(*body).readLocked
         0     0% 29.58%    -904640  0.23%  net/http.(*response).WriteHeader
         0     0% 29.58% -118725478 29.53%  net/http.HandlerFunc.ServeHTTP
         0     0% 29.58%    -950286  0.24%  net/http.Header.Get (partial-inline)
         0     0% 29.58% -118398646 29.45%  net/http.serverHandler.ServeHTTP
         0     0% 29.58%    -753675  0.19%  net/textproto.CanonicalMIMEHeaderKey
         0     0% 29.58%    -950286  0.24%  net/textproto.MIMEHeader.Get
         0     0% 29.58%     589833  0.15%  reflect.Value.Interface (inline)
         0     0% 29.58%     589833  0.15%  reflect.packEface (inline)
         0     0% 29.58%     589833  0.15%  reflect.packEfaceData
         0     0% 29.58%     589833  0.15%  reflect.valueInterface
         0     0% 29.58%    4571209  1.14%  strings.Split (inline)


go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
File: main
Build ID: 4fc71448bf5b904075b7d7b9df3fe4a1e28fdc5e
Type: inuse_space
Time: 2026-10-11 19:16:07 MSK
Showing nodes accounting for 6839.32kB, 130.10% of 5256.91kB total
Dropped 8 nodes (cum <= 26.28kB)
      flat  flat%   sum%        cum   cum%
 3598.02kB 68.44% 68.44%  3598.02kB 68.44%  bufio.NewWriterSize (inline)
 1576.28kB 29.99% 98.43%  1576.28kB 29.99%  github.com/jackc/pgx/v5/internal/stmtcache.NewLRUCache (inline)
    1028kB 19.56% 117.98%     1028kB 19.56%  bufio.NewReaderSize (inline)
 -902.59kB 17.17% 100.81%  -902.59kB 17.17%  compress/flate.NewWriter (inline)
  514.63kB  9.79% 110.60%   514.63kB  9.79%  github.com/jackc/pgx/v5/pgxpool.(*connResource).getPoolRow (inline)
  513.31kB  9.76% 120.37%  1064.41kB 20.25%  github.com/jackc/pgx/v5/pgxpool.NewWithConfig.func3
 -512.69kB  9.75% 110.62%  -512.69kB  9.75%  github.com/jackc/pgx/v5/pgconn.connectOne
 -512.50kB  9.75% 100.87%   551.10kB 10.48%  github.com/jackc/pgx/v5.connect
  512.50kB  9.75% 110.62%   512.50kB  9.75%  net/http.(*Transport).queueForIdleConn
  512.23kB  9.74% 120.36%   512.23kB  9.74%  runtime.mallocgc
  512.20kB  9.74% 130.10%   512.20kB  9.74%  github.com/jackc/pgx/v5/pgtype.(*Map).planEncodeDepth
 -512.14kB  9.74% 120.36%  -512.14kB  9.74%  github.com/jackc/pgx/v5.(*Conn).getRows
  512.04kB  9.74% 130.10%   512.04kB  9.74%  crypto/internal/fips140/ecdh.init
  512.04kB  9.74% 139.84%   512.04kB  9.74%  internal/bytealg.MakeNoZero
 -512.03kB  9.74% 130.10%  -512.03kB  9.74%  github.com/jackc/pgx/v5/internal/stmtcache.StatementName
         0     0% 130.10%      514kB  9.78%  bufio.NewReader (inline)
         0     0% 130.10%  -902.59kB 17.17%  compress/gzip.(*Writer).Close
         0     0% 130.10%  -902.59kB 17.17%  compress/gzip.(*Writer).Write
         0     0% 130.10%  -899.93kB 17.12%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 130.10%  1026.83kB 19.53%  github.com/go-chi/chi/v5/middleware.(*Compressor).Handler-fm.(*Compressor).Handler.func1
         0     0% 130.10%   512.04kB  9.74%  github.com/golang-migrate/migrate/v4.New
         0     0% 130.10%   512.04kB  9.74%  github.com/golang-migrate/migrate/v4/database.Open
         0     0% 130.10%   512.04kB  9.74%  github.com/golang-migrate/migrate/v4/database/postgres.(*Postgres).Open
         0     0% 130.10%   512.50kB  9.75%  github.com/hashicorp/go-retryablehttp.(*Client).Do
         0     0% 130.10%  -512.03kB  9.74%  github.com/jackc/pgx/v5.(*Conn).Exec
         0     0% 130.10%  -512.03kB  9.74%  github.com/jackc/pgx/v5.(*Conn).exec
         0     0% 130.10%   512.20kB  9.74%  github.com/jackc/pgx/v5.(*ExtendedQueryBuilder).Build
         0     0% 130.10%   512.20kB  9.74%  github.com/jackc/pgx/v5.(*ExtendedQueryBuilder).appendParam
         0     0% 130.10%   512.20kB  9.74%  github.com/jackc/pgx/v5.(*ExtendedQueryBuilder).encodeExtendedParamValue
         0     0% 130.10%   551.10kB 10.48%  github.com/jackc/pgx/v5.ConnectConfig
         0     0% 130.10%  -512.69kB  9.75%  github.com/jackc/pgx/v5/pgconn.ConnectConfig
         0     0% 130.10%  -512.69kB  9.75%  github.com/jackc/pgx/v5/pgconn.connectPreferred
         0     0% 130.10%   512.20kB  9.74%  github.com/jackc/pgx/v5/pgtype.(*Map).Encode
         0     0% 130.10%   512.20kB  9.74%  github.com/jackc/pgx/v5/pgtype.(*Map).PlanEncode (inline)
         0     0% 130.10%  -512.03kB  9.74%  github.com/jackc/pgx/v5/pgxpool.(*Conn).Exec
         0     0% 130.10%   514.63kB  9.79%  github.com/jackc/pgx/v5/pgxpool.(*Conn).getPoolRow (inline)
         0     0% 130.10%  -512.03kB  9.74%  github.com/jackc/pgx/v5/pgxpool.(*Pool).Exec
         0     0% 130.10%   514.69kB  9.79%  github.com/jackc/pgx/v5/pgxpool.(*Pool).QueryRow
         0     0% 130.10%  1064.41kB 20.25%  github.com/jackc/puddle/v2.(*Pool[go.shape.*uint8]).initResourceValue.func1
         0     0% 130.10%   512.50kB  9.75%  github.com/mersikovs/korob.git/internal/audit.(*Dispatcher).processEvent
         0     0% 130.10%   512.50kB  9.75%  github.com/mersikovs/korob.git/internal/audit.(*Dispatcher).processEvent.func1
         0     0% 130.10%   512.50kB  9.75%  github.com/mersikovs/korob.git/internal/audit.(*Dispatcher).startWorkerPool.func1
         0     0% 130.10%   512.50kB  9.75%  github.com/mersikovs/korob.git/internal/audit.(*RemoteObserver).Notify
         0     0% 130.10%   512.04kB  9.74%  github.com/mersikovs/korob.git/internal/database.MigrateUp
         0     0% 130.10%  -512.03kB  9.74%  github.com/mersikovs/korob.git/internal/handler.(*MetricHandler).BatchUpdateMetricsFromBody
         0     0% 130.10%   514.63kB  9.79%  github.com/mersikovs/korob.git/internal/handler.(*MetricHandler).UpdateMetricFromPath
         0     0% 130.10%  -902.59kB 17.17%  github.com/mersikovs/korob.git/internal/middleware.(*compressWriter).Close (inline)
         0     0% 130.10% -1024.17kB 19.48%  github.com/mersikovs/korob.git/internal/middleware.GzipRequestMiddleware.func1
         0     0% 130.10% -1926.76kB 36.65%  github.com/mersikovs/korob.git/internal/middleware.GzipResponseMiddleware.func1
         0     0% 130.10%  -902.59kB 17.17%  github.com/mersikovs/korob.git/internal/middleware.GzipResponseMiddleware.func1.1
         0     0% 130.10%  -512.03kB  9.74%  github.com/mersikovs/korob.git/internal/repository.(*PgStorage).BatchSave
         0     0% 130.10%   514.69kB  9.79%  github.com/mersikovs/korob.git/internal/repository.(*PgStorage).Get
         0     0% 130.10%  -899.93kB 17.12%  github.com/mersikovs/korob.git/internal/router.NewRouter.Logger.func1.1
         0     0% 130.10%   514.63kB  9.79%  github.com/mersikovs/korob.git/internal/service.(*MetricService).UpdateMetric
         0     0% 130.10%  -512.03kB  9.74%  github.com/mersikovs/korob.git/internal/service.(*MetricService).UpdateMetricsFromStruct
         0     0% 130.10%   512.04kB  9.74%  main.main
         0     0% 130.10%   512.50kB  9.75%  net/http.(*Client).Do (inline)
         0     0% 130.10%   512.50kB  9.75%  net/http.(*Client).do
         0     0% 130.10%   512.50kB  9.75%  net/http.(*Client).send
         0     0% 130.10%   512.50kB  9.75%  net/http.(*Transport).RoundTrip
         0     0% 130.10%      514kB  9.78%  net/http.(*Transport).dialConn
         0     0% 130.10%      514kB  9.78%  net/http.(*Transport).dialConnFor
         0     0% 130.10%   512.50kB  9.75%  net/http.(*Transport).getConn
         0     0% 130.10%   512.50kB  9.75%  net/http.(*Transport).roundTrip
         0     0% 130.10%      514kB  9.78%  net/http.(*Transport).startDialConnForLocked.func1
         0     0% 130.10%  3212.09kB 61.10%  net/http.(*conn).serve
         0     0% 130.10%  -899.93kB 17.12%  net/http.HandlerFunc.ServeHTTP
         0     0% 130.10%      514kB  9.78%  net/http.newBufioReader
         0     0% 130.10%  3598.02kB 68.44%  net/http.newBufioWriterSize
         0     0% 130.10%   512.50kB  9.75%  net/http.send
         0     0% 130.10%  -899.93kB 17.12%  net/http.serverHandler.ServeHTTP
         0     0% 130.10%   512.04kB  9.74%  net/url.(*URL).String
         0     0% 130.10%   512.04kB  9.74%  runtime.doInit (inline)
         0     0% 130.10%   512.04kB  9.74%  runtime.doInit1
         0     0% 130.10%  1024.08kB 19.48%  runtime.main
         0     0% 130.10%   512.23kB  9.74%  runtime.malg
         0     0% 130.10%   512.23kB  9.74%  runtime.newobject
         0     0% 130.10%   512.23kB  9.74%  runtime.newproc.func1
         0     0% 130.10%   512.23kB  9.74%  runtime.newproc1
         0     0% 130.10%   512.23kB  9.74%  runtime.systemstack
         0     0% 130.10%   512.04kB  9.74%  strings.(*Builder).Grow
         0     0% 130.10%   512.04kB  9.74%  strings.(*Builder).grow
         0     0% 130.10%   512.50kB  9.75%  sync.(*WaitGroup).Go.func1