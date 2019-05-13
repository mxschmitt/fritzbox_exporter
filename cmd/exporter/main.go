package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	envstruct "github.com/mxschmitt/golang-env-struct"

	"github.com/mxschmitt/fritzbox_exporter/pkg/fritzboxmetrics"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type Settings struct {
	Stdout     bool   `env:"STDOUT"`
	ListenAddr string `env:"LISTEN_ADDR"`
	FritzBox   struct {
		IP       string `env:"IP"`
		Port     int    `env:"PORT"`
		UserName string `env:"USERNAME"`
		Password string `env:"PASSWORD"`
	} `env:"FRITZ_BOX"`
}

func main() {
	log.SetFlags(log.Llongfile)

	settings := &Settings{}
	flag.BoolVar(&settings.Stdout, "stdout", false, "print all available metrics to stdout")
	flag.StringVar(&settings.ListenAddr, "listen-address", ":9133", "The address to listen on for HTTP requests.")

	flag.StringVar(&settings.FritzBox.IP, "gateway-address", "fritz.box", "The hostname or IP of the FRITZ!Box")
	flag.IntVar(&settings.FritzBox.Port, "gateway-port", 49000, "The port of the FRITZ!Box UPnP service")
	flag.StringVar(&settings.FritzBox.UserName, "username", "", "The user for the FRITZ!Box UPnP service")
	flag.StringVar(&settings.FritzBox.Password, "password", "", "The password for the FRITZ!Box UPnP service")
	flag.Parse()

	if err := envstruct.ApplyEnvVars(settings, "FRITZ_BOX_EXPORTER"); err != nil {
		log.Fatalf("could not apply environment variables: %v", err)
	}

	if settings.Stdout {
		if err := printToStdout(settings); err != nil {
			log.Fatalf("could not print metrics to stdout: %v", err)
		}
		return
	}

	collector := &FritzboxCollector{
		Gateway:  settings.FritzBox.IP,
		Port:     uint16(settings.FritzBox.Port),
		Username: settings.FritzBox.UserName,
		Password: settings.FritzBox.Password,
	}

	go collector.LoadServices()

	prometheus.MustRegister(collector)
	prometheus.MustRegister(collectErrors)

	http.Handle("/metrics", prometheus.Handler())
	log.Fatal(http.ListenAndServe(settings.ListenAddr, nil))
}

func printToStdout(settings *Settings) error {
	root, err := fritzboxmetrics.LoadServices(settings.FritzBox.IP, uint16(settings.FritzBox.Port), settings.FritzBox.UserName, settings.FritzBox.IP)
	if err != nil {
		return errors.Wrap(err, "could not load UPnP service")
	}

	for _, s := range root.Services {
		for _, a := range s.Actions {
			if !a.IsGetOnly() {
				continue
			}

			res, err := a.Call()
			if err != nil {
				log.Printf("unexpected error for action %s: %v", a.Name, err)
				continue
			}

			fmt.Printf("  %s\n", a.Name)
			for _, arg := range a.Arguments {
				fmt.Printf("    %s: %v\n", arg.RelatedStateVariable, res[arg.StateVariable.Name])
			}
		}
	}
	return nil
}
