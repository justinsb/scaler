/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	clientset "github.com/justinsb/scaler/pkg/client/clientset/versioned"
	informers "github.com/justinsb/scaler/pkg/client/informers/externalversions"
	"github.com/justinsb/scaler/pkg/control"
	"github.com/justinsb/scaler/pkg/control/target"
	"github.com/justinsb/scaler/pkg/http"
	"github.com/justinsb/scaler/pkg/signals"
	"github.com/justinsb/scaler/pkg/version"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/clock"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config := options.NewAutoScalerConfig()
	config.AddFlags(pflag.CommandLine)
	config.InitFlags()

	if config.PrintVersion {
		fmt.Printf("%s\n", version.VERSION)
		os.Exit(0)
	}

	// Perform further validation of flags.
	if err := config.ValidateFlags(); err != nil {
		glog.Errorf("%v", err)
		os.Exit(1)
	}

	err := run(config)
	if err != nil {
		glog.Errorf("%v", err)
		os.Exit(1)
	}
}

func run(config *options.AutoScalerConfig) error {
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	var cfg *rest.Config
	var err error
	if config.Kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return err
	}
	// Use protobufs for communication with apiserver.
	// But don't use them for our CRD - it doesn't work.

	scalingClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("error building scaling client: %v", err)
	}

	cfg.ContentType = "application/vnd.kubernetes.protobuf"
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	// TODO: Are these resync times way too low?

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	scalerInformerFactory := informers.NewSharedInformerFactory(scalingClient, time.Second*30)

	t, err := target.NewKubernetesTarget(kubeClient)
	if err != nil {
		return err
	}

	state, err := control.NewState(&clock.RealClock{}, t, config)
	if err != nil {
		return fmt.Errorf("error initializing: %v", err)
	}

	controller, err := control.NewController(state, kubeClient, scalingClient, kubeInformerFactory, scalerInformerFactory)
	if err != nil {
		return fmt.Errorf("error building controller: %v", err)
	}

	go kubeInformerFactory.Start(stopCh)
	go scalerInformerFactory.Start(stopCh)

	if config.ListenAPI != "" {
		server, err := http.NewAPIServer(config, state)
		if err != nil {
			return fmt.Errorf("error creating APIServer: %v", err)
		}
		go func() {
			err := server.Start(stopCh)
			if err != nil {
				glog.Fatalf("error starting APIServer: %v", err)
			}
		}()
	}

	if err = controller.Run(2, stopCh); err != nil {
		return err
	}

	return nil
}
