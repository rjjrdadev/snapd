// -*- Mode: Go; indent-tabs-mode: t -*-
// +build !excludeintegration

/*
 * Copyright (C) 2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package store

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"gopkg.in/tylerb/graceful.v1"

	"github.com/ubuntu-core/snappy/snap"
)

// Store is our snappy software store implementation
type Store struct {
	url     string
	blobDir string

	srv *graceful.Server

	snaps map[string]string
}

var defaultAddr = "localhost:11028"

func rootEndpoint(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(418)
	fmt.Fprintf(w, "I'm a teapot")
}

func searchEndpoint(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(501)
	fmt.Fprintf(w, "search not implemented yet")
}

func detailsEndpoint(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(501)
	fmt.Fprintf(w, "details not implemented yet")
}

func bulkEndpoint(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(501)
	fmt.Fprintf(w, "bulk not implemented yet")
}

// NewStore creates a new store server
func NewStore(blobDir string) *Store {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootEndpoint)
	mux.HandleFunc("/search", searchEndpoint)
	mux.HandleFunc("/package/", detailsEndpoint)
	mux.HandleFunc("/click-metadata", bulkEndpoint)
	mux.Handle("/download/", http.StripPrefix("/download/", http.FileServer(http.Dir(blobDir))))

	store := &Store{
		blobDir: blobDir,
		snaps:   make(map[string]string),

		url: fmt.Sprintf("http://%s", defaultAddr),
		srv: &graceful.Server{
			Timeout: 2 * time.Second,

			Server: &http.Server{
				Addr:    defaultAddr,
				Handler: mux,
			},
		},
	}

	return store
}

// URL returns the base-url that the store is listening on
func (s *Store) URL() string {
	return s.url
}

// Start listening
func (s *Store) Start() error {
	l, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}

	go s.srv.Serve(l)
	return nil
}

// Stop stops the server
func (s *Store) Stop() error {
	s.srv.Stop(0)
	timeoutTime := 2000 * time.Millisecond

	select {
	case <-s.srv.StopChan():
	case <-time.After(timeoutTime):
		return fmt.Errorf("store failed to stop after %s", timeoutTime)
	}

	return nil
}

func (s *Store) refreshSnaps() error {
	snaps, err := filepath.Glob(filepath.Join(s.blobDir, "*.snap"))
	if err != nil {
		return err
	}

	for _, fn := range snaps {
		snapFile, err := snap.Open(fn)
		if err != nil {
			return err
		}
		info, err := snapFile.Info()
		if err != nil {
			return err
		}
		s.snaps[info.Name] = fn
	}

	return nil
}
