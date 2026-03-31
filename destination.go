package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

func checkDestination(requestCtx context.Context, targetURL string) error {
	_, span := tracer.Start(requestCtx, "http.verify_destination")
	defer span.End()
	resp, err := http.DefaultClient.Get(targetURL)
	if err != nil {
		return fmt.Errorf("destination unreachable: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("destination returned status %d", resp.StatusCode)
	}
	return nil
}
