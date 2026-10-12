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