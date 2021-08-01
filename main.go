package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/mux"
)

type producer struct {
	topic string
	body  []byte
}

var kafkaprod *kafka.Producer

func main() {
	cpus := runtime.NumCPU()
	if cpus == 1 {
		cpus = 1
	} else {
		cpus = cpus - 1
	}
	runtime.GOMAXPROCS(cpus)
	f, _ := os.OpenFile("server.logs", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	log.SetOutput(f)
	var p producer
	var err error
	kafkaprod, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost", "debug": "broker"})
	if err != nil {
		log.Fatal("kafka producer not alive with error", err)
	}
	defer kafkaprod.Close()
	go p.deliveryNotifications()
	r := mux.NewRouter()
	r.HandleFunc("/{topic}", p.producerHandler).Methods("POST")
	r.HandleFunc("/wap/{topic}", p.producerHandler).Methods("POST")
	log.Println("starting server")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// if err := http.ListenAndServeTLS(":8090", "/wap/software/certs/bngrenew.crt", "/wap/software/certs/bngrenew_key.txt", r); err != nil {
		//  log.Println("server crashed ", err)
		//  wg.Done()
		// }
		if err := http.ListenAndServe(":8082", r); err != nil {
			log.Println("server crashed", err)
			wg.Done()
		}
	}()
	fmt.Println("server running successfully")
	wg.Wait()
	kafkaprod.Flush(10 * 1000)
}

//listen to delivery notifications on channel kafkaprod.Events()
func (p *producer) deliveryNotifications() {
	for e := range kafkaprod.Events() {
		switch ev := e.(type) {
		case error:
			//error occurred in delivery of message,
			//therefore open the topic file and write the log in the file
			log.Println("error writing message to topic delivery notification", e.String())
			writeToFile(p.topic, p.body)
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				writeToFile(p.topic, p.body)
			}
		}
	}
}

// producerHandler is the handler that will handle http requests for
func (p producer) producerHandler(w http.ResponseWriter, r *http.Request) {
	p.topic = mux.Vars(r)["topic"]
	var err error
	p.body, err = ioutil.ReadAll(r.Body)
	//log.Println("body of request ",string(p.body))
	if err != nil {
		log.Println("error while reading message body ", err)
	}
	err = kafkaprod.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
		Value:          p.body,
	}, nil)
	if err != nil {
		log.Println("error producing", err)
		writeToFile(p.topic, p.body)
	}
}

// writeToFile writes the contents to a file.
func writeToFile(name string, body []byte) {
	f, err := os.OpenFile(fmt.Sprintf("%v.logs", name), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	defer f.Close()
	if err != nil {
		log.Println("unable to write the log ")
		log.Println(string(body))
		return
	}
	f.Write(body)
	f.Write([]byte("\n"))
	// go closeAndCreateNewP()
	return
}
