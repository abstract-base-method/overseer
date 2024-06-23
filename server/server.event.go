package server

import (
	"context"
	"fmt"
	"io"
	v1 "overseer/build/go"
	"overseer/common"
	"overseer/engine"

	charm "github.com/charmbracelet/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type defaultEventServer struct {
	bus engine.EventBus
	log *charm.Logger
	v1.UnimplementedEventsServer
}

func NewEventServer(bus engine.EventBus) v1.EventsServer {
	return &defaultEventServer{
		bus: bus,
		log: common.GetLogger("server.event"),
	}
}

func (s *defaultEventServer) GetEvent(ctx context.Context, req *v1.GetEventRequest) (*v1.EventRecord, error) {
	return nil, status.Error(codes.Unimplemented, "method GetEvent not implemented")
}

func (s *defaultEventServer) Submit(ctx context.Context, event *v1.Event) (*v1.EventReceipts, error) {
	info, err := common.GetContextInformation(ctx)
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return nil, err
	}

	s.log.Debug("submitting event", info.LoggingContext("event", event)...)
	results, err := s.bus.Submit(ctx, event)
	if err != nil {
		s.log.Error("failed to submit event", info.LoggingContext("error", err)...)
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to submit event: %v", err))
	}

	// collect all results from channel until it closes then return
	receipts := make([]*v1.EventReceipt, 0)
	for result := range results {
		s.log.Debug("got result", info.LoggingContext("result", result)...)
		receipts = append(receipts, result)
	}

	s.log.Debug("all results received", info.LoggingContext("count", len(receipts))...)
	return &v1.EventReceipts{Receipts: receipts}, nil
}

func (s *defaultEventServer) Subscribe(stream v1.Events_SubscribeServer) error {
	info, err := common.GetContextInformation(stream.Context())
	if err != nil {
		s.log.Error("failed to get context information", "error", err)
		return err
	}
	s.log.Info("client subscribed", info.LoggingContext()...)
	for {
		event, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				s.log.Info("got EOF from client", info.LoggingContext()...)
				return nil
			}
			s.log.Error("failed to recieve event from client", info.LoggingContext("error", err)...)
			return status.Error(codes.Internal, "failed to recieve event from client")
		}
		results, err := s.bus.Submit(stream.Context(), event)
		if err != nil {
			s.log.Error("failed to submit event", info.LoggingContext("error", err)...)
			return status.Error(codes.Internal, "failed to submit event")
		}
		go s.transmitResults(stream, results)
	}
}

func (s *defaultEventServer) transmitResults(stream v1.Events_SubscribeServer, results <-chan *v1.EventReceipt) {
	info, _ := common.GetContextInformation(stream.Context())
	for {
		select {
		case <-stream.Context().Done():
			s.log.Debug("stream context done", info.LoggingContext()...)
			return
		case result, ok := <-results:
			if !ok {
				s.log.Debug("results channel closed", info.LoggingContext()...)
				return
			}
			if err := stream.Send(result); err != nil {
				s.log.Error("failed to send result to client", info.LoggingContext("error", err)...)
				return
			}
		}
	}
}
