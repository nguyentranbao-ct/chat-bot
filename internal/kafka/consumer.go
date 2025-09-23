package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	httpkit "github.com/carousell/ct-go/pkg/httpclient"
	"github.com/carousell/ct-go/pkg/json"
	"github.com/carousell/ct-go/pkg/logger"
	log "github.com/carousell/ct-go/pkg/logger/log_context"
	"github.com/carousell/ct-go/pkg/workerpool"
	"github.com/nguyentranbao-ct/chat-bot/pkg/ctxval"
	"github.com/nguyentranbao-ct/chat-bot/pkg/tmplx"
	"github.com/nguyentranbao-ct/chat-bot/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/kafka-go"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type kafkaConsumer struct {
	reader  *kafka.Reader
	metrics *prometheus.HistogramVec
	opts    consumerOptions
}

type consumerOptions struct {
	sd             fx.Shutdowner
	lc             fx.Lifecycle
	readerConf     kafka.ReaderConfig
	maxWorkers     int
	consumeTimeout time.Duration
	handler        func(context.Context, kafka.Message) error
}

func startKafkaConsumer(opts consumerOptions) error {
	metrics, err := util.GetHistogramVec("kafka_messages_consumed", "status", "topic", "group")
	if err != nil {
		return fmt.Errorf("get histogram vec: %w", err)
	}
	worker := &kafkaConsumer{
		reader:  kafka.NewReader(opts.readerConf),
		metrics: metrics,
		opts:    opts,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	opts.lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				log.Infow(ctx, "start consuming...", "topics", opts.readerConf.GroupTopics)
				if err := worker.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
					log.Errorw(ctx, "consume jobs", "error", err)
				}
				done <- errors.Join(err, opts.sd.Shutdown())
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			log.Warnf(ctx, "shutting down...")
			cancel()
			// <-done
			return nil
		},
	})

	return nil
}

func (w *kafkaConsumer) Start(ctx context.Context) error {
	defer w.reader.Close()

	pool := workerpool.New(w.opts.maxWorkers)
	defer pool.Close()

	groupID := w.reader.Config().GroupID
	for ctx.Err() == nil {
		msg, err := w.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		pool.Run(func() {
			start := time.Now()
			lagMs := start.Sub(msg.Time).Milliseconds()

			ctx := httpkit.InjectCorrelationIDToContext(ctx, httpkit.GenerateCorrelationID())
			ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), w.opts.consumeTimeout)
			defer cancel()

			// wrap context with shared values
			ctx = ctxval.Wrap(ctx)

			err := w.opts.handler(ctx, msg)
			duration := time.Since(start)

			code := getCode(err)
			level := getLogLevel(code)
			log.Logw(ctx, level, "kafka",
				"code", code,
				"duration_ms", duration.Milliseconds(),
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"lag_ms", lagMs,
				"key", string(msg.Key),
				"value", json.RawMessage(msg.Value),
				logger.Error(err),
			)
			if w.metrics != nil {
				w.metrics.
					WithLabelValues(code.String(), msg.Topic, groupID).
					Observe(duration.Seconds())
			}
		})
	}
	return nil
}

func getCode(err error) codes.Code {
	if errors.Is(err, context.DeadlineExceeded) {
		return codes.DeadlineExceeded
	}
	if errors.Is(err, context.Canceled) {
		return codes.Canceled
	}
	if errors.Is(err, tmplx.ErrParseTemplate) {
		return codes.InvalidArgument
	}
	if errors.Is(err, tmplx.ErrRenderTemplate) {
		return codes.InvalidArgument
	}
	st, ok := status.FromError(err)
	if !ok {
		return status.Code(errors.Unwrap(err))
	}
	return st.Code()
}

func getLogLevel(code codes.Code) logger.Level {
	switch code {
	case codes.OK:
		return logger.InfoLevel
	case codes.Canceled,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.Unimplemented,
		codes.OutOfRange:
		return logger.WarnLevel
	default:
		return logger.ErrorLevel
	}
}
