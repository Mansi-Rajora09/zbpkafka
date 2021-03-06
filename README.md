# zbpkafka
STEP 1

vim ~/.bashrc

alias zbpstop='sudo systemctl stop zbp'
alias tailzbp='sudo journalctl -fu zbp'
alias zbpres='sudo systemctl restart zbp'
alias zbpstatus='sudo systemctl status zbp'

. ~/.bashrc

STEP 2 =====================


sudo vim zbp.service
 less /etc/systemd/system/zbp.service

[Unit]
Description=zbp server

[Service]
User=bizops
WorkingDirectory=/home/bizops/test
ExecStart=/home/bizops/test/main
Restart=always
sudosudzbp

[Install]
WantedBy=multi-user.target

--------
STEP 3===============

Under /home/bizops/test
Vi .env

DB_USER='cdr'
DB_PASSWORD='cdr'
# 1 for loan only, 2 for loan and vas both
LOANANDVAS='1'

LOGLEVEL=DEBUG

DEFAULTLANG='_E'

# 1 for repeat 2 for end, to be done in the end
REPEAT='1'

STEP 4 

=================
On terminal run these commands :
zbpstatus
zbpres

curl 'localhost:7273/zbp/getOptions?shortcode=265&actionId=-1&apartymsisdn=111277958244&userinput=1'

STEP 5
Go to the zbp database and run
Select * from ivr_main_menu
Check the url column and curl it 
http://192.168.41.77/zbploan/index.php?requestType=eligibilityCheck&msisdn=$msisdn
curl 'http://192.168.41.77/zbploan/index.php?requestType=eligibilityCheck&msisdn=111277958244'
If you don’t get any response change the url 
update ivr_main_menu set  url ='http://192.168.41.77/zbploan/index.php?requestType=eligib
ilityCheck&msisdn=$msisdn' where  id =1;




------Kafka Module Step s------------

Setup is already there in Rwanda     ----ssh bizops@10.7.7.51

Steps for kafka Module 
Step 1: copy the main.go to any path and run go build -o main main.go
Step 2: vi /etc/systemd/system/wap.service and paste   957  sudo su

-------
[Unit]
Description=wap service logs server

[Service]
User=bizops
WorkingDirectory=/home/bizops/downloads/logsServer
ExecStart=/home/bizops/downloads/logsServer/main
Restart=always

[Install]
WantedBy=multi-user.target

---------
Step 3 : run sudo systemctl daemon-reload
Step 4: sudo systemctl restart wap
Step 5 : Create a topic with name as zbp 

 ./kafka-topics.sh -zookeeper localhost:2181 -replication-factor 1 -partitions 1 -topic zbp -create

Step 6 : send a post request on that topic 

curl -i -X POST -H "Content-Type: application/json" -d "{\"key\":\"val\"}" http://localhost:8082/zbp

Step 7: Consume that topic from producer 

./kafka-console-consumer.sh --bootstrap-server localhost:9092 -topic zbp --from-beginning















=======================================================================================================================================================================================================================================================================================================================================================================================main.go  File ========


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





=======================================================================================================================================================================================================================================================================================================================================================================================main.go  File END ========
