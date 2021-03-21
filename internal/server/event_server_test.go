/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/sethvargo/go-limiter/httplimit"
	"github.com/sethvargo/go-limiter/memorystore"
	corev1 "k8s.io/api/core/v1"

	"github.com/fluxcd/pkg/recorder"
)

func TestEventKeyFunc(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup middleware
	store, err := memorystore.New(&memorystore.Config{
		Interval: 10 * time.Minute,
	})
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	middleware, err := httplimit.NewMiddleware(store, eventKeyFunc)
	g.Expect(err).ShouldNot(gomega.HaveOccurred())
	handler := middleware.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make request
	tests := []struct {
		involvedObject corev1.ObjectReference
		severity       string
		rateLimit      bool
	}{
		{
			involvedObject: corev1.ObjectReference{
				APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
				Kind:       "Kustomization",
				Name:       "1",
				Namespace:  "1",
			},
			severity:  recorder.EventSeverityInfo,
			rateLimit: false,
		},
		{
			involvedObject: corev1.ObjectReference{
				APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
				Kind:       "Kustomization",
				Name:       "1",
				Namespace:  "1",
			},
			severity:  recorder.EventSeverityInfo,
			rateLimit: true,
		},
		{
			involvedObject: corev1.ObjectReference{
				APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
				Kind:       "Kustomization",
				Name:       "1",
				Namespace:  "1",
			},
			severity:  recorder.EventSeverityError,
			rateLimit: false,
		},
		{
			involvedObject: corev1.ObjectReference{
				APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
				Kind:       "Kustomization",
				Name:       "1",
				Namespace:  "1",
			},
			severity:  recorder.EventSeverityInfo,
			rateLimit: false,
		},
		{
			involvedObject: corev1.ObjectReference{
				APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
				Kind:       "Kustomization",
				Name:       "2",
				Namespace:  "2",
			},
			severity:  recorder.EventSeverityInfo,
			rateLimit: false,
		},
		{
			involvedObject: corev1.ObjectReference{
				APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
				Kind:       "Kustomization",
				Name:       "3",
				Namespace:  "3",
			},
			severity:  recorder.EventSeverityInfo,
			rateLimit: false,
		},
		{
			involvedObject: corev1.ObjectReference{
				APIVersion: "kustomize.toolkit.fluxcd.io/v1beta1",
				Kind:       "Kustomization",
				Name:       "2",
				Namespace:  "2",
			},
			severity:  recorder.EventSeverityInfo,
			rateLimit: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.involvedObject.String(), func(t *testing.T) {
			event := recorder.Event{
				InvolvedObject: tt.involvedObject,
				Severity:       tt.severity,
			}
			eventData, err := json.Marshal(event)
			g.Expect(err).ShouldNot(gomega.HaveOccurred())

			req := httptest.NewRequest("POST", "/", bytes.NewBuffer(eventData))
			g.Expect(err).ShouldNot(gomega.HaveOccurred())
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			if tt.rateLimit {
				g.Expect(res.Code).Should(gomega.Equal(429))
				g.Expect(res.Header().Get("X-Ratelimit-Remaining")).Should(gomega.Equal("0"))
			} else {
				g.Expect(res.Code).Should(gomega.Equal(200))
			}
		})
	}
}
