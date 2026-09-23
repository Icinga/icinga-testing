package services

import (
	"context"
	"net/http"
	"time"
)

// IcingaNotificationsWebhookReceiver is a minimal HTTP web server for the Icinga Notifications Webhook channel.
//
// After being launched, bound on the host to the Docker Network's "Gateway" IPv4 address, incoming requests will be
// passed to the Handler http.HandlerFunc which MUST be set to the custom receiver.
type IcingaNotificationsWebhookReceiver struct {
	ListenAddr string
	Handler    http.HandlerFunc
	server     *http.Server
}

// LaunchIcingaNotificationsWebhookReceiver starts an IcingaNotificationsWebhookReceiver's webserver on the listen address.
func LaunchIcingaNotificationsWebhookReceiver(listen string) (*IcingaNotificationsWebhookReceiver, error) {
	webhookRec := &IcingaNotificationsWebhookReceiver{
		ListenAddr: listen,
		Handler: func(writer http.ResponseWriter, request *http.Request) {
			// Default handler to not run into nil pointer dereference errors.
			_ = request.Body.Close()
			http.Error(writer, "¯\\_(ツ)_/¯", http.StatusServiceUnavailable)
		},
	}
	webhookRec.server = &http.Server{
		Addr:    listen,
		Handler: &webhookRec.Handler,
	}

	errCh := make(chan error)
	go func() { errCh <- webhookRec.server.ListenAndServe() }()

	select {
	case err := <-errCh:
		return nil, err
	case <-time.After(time.Second):
		return webhookRec, nil
	}
}

// Cleanup closes both the web server and the Requests channel.
func (webhookRec *IcingaNotificationsWebhookReceiver) Cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = webhookRec.server.Shutdown(ctx)
}
