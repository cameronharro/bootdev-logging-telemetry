package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	pkgerr "github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

var allowedUsers = map[string]string{
	"frodo":   "$2a$10$B6O/n6teuCzpuh66jrUAdeaJ3WvXcxRkzpN0x7H.di9G9e/NGb9Me",
	"samwise": "$2a$10$EWZpvYhUJtJcEMmm/IBOsOGIcpxUnGIVMRiDlN/nxl1RRwWGkJtty",
	// frodo: "ofTheNineFingers"
	// samwise: "theStrong"
	"saruman": "invalidFormat",
}

func (s *server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			httpError(r.Context(), w, errors.New("Unauthorized"), http.StatusUnauthorized)
			return
		}
		stored, exists := allowedUsers[username]
		if !exists {
			httpError(r.Context(), w, errors.New("Unauthorized"), http.StatusUnauthorized)
			return
		}
		ok, err := s.validatePassword(r.Context(), password, stored)
		if err != nil {
			httpError(r.Context(), w, fmt.Errorf("Internal Server Error: %w", err), http.StatusInternalServerError)
			return
		}
		if !ok {
			httpError(r.Context(), w, errors.New("Unauthorized"), http.StatusUnauthorized)
			return
		}
		if logContext, ok := r.Context().Value(logContextKey).(*LogContext); ok && logContext != nil {
			logContext.Username = username
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) validatePassword(requestCtx context.Context, password, stored string) (bool, error) {
	_, span := tracer.Start(requestCtx, "auth.validate_password")
	defer span.End()
	err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	if err != nil {
		return false, pkgerr.WithStack(err)
	}
	return true, nil
}
