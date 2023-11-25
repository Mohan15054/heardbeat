package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/cpu"
)

var (
	topic                   string
	protocol                string
	broker                  string
	portStr                 string
	client                  mqtt.Client
	mqtt_heardbeat_interval int
	elapsedMilliseconds     int64
	interval                int64
)

// Message represents the structure of your JSON payload
type Message struct {
	IID   string  `json:"iid"`
	Key   string  `json:"key"`
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	// Handle incoming messages if needed
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	var broker = os.Getenv("MQTT_BROKER")
	portStr := os.Getenv("MQTT_PORT")
	port, err := strconv.Atoi(portStr)

	if err != nil {
		log.Fatal("Error loading .env file")
	}
	insecureSkipVerifyStr := os.Getenv("MQTT_TLS_INSECURE_SKIP_VERIFY")
	insecureSkipVerify, err := strconv.ParseBool(insecureSkipVerifyStr)

	if err != nil {
		log.Fatalln(err.Error())
	}

	protocol = os.Getenv("MQTT_PROTOCOL")
	MQTT_CLIENT_ID := os.Getenv("MQTT_CLIENT_ID")
	mqtt_usr := os.Getenv("MQTT_USERNAME")
	mqtt_pass := os.Getenv("MQTT_PASSWORD")
	topic = os.Getenv("MQTT_TOPIC")
	mqtt_time_format := os.Getenv("MQTT_TIME_FORMAT")

	fmt.Printf("Connect to %s://%s:%d\n", protocol, broker, port)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", protocol, broker, port))
	tlsConfig := NewTlsConfig()
	tlsConfig.InsecureSkipVerify = insecureSkipVerify
	opts.SetTLSConfig(tlsConfig)

	opts.SetClientID(MQTT_CLIENT_ID + time.Now().Format(mqtt_time_format))
	opts.SetUsername(mqtt_usr)
	opts.SetPassword(mqtt_pass)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	for {
		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
			time.Sleep(5 * time.Second)
		} else {
			fmt.Println("Connected to MQTT broker")
			defer client.Disconnect(250)
			go publishCPULoad(client)

			select {}
			break
		}
	}

}
func publishCPULoad(client mqtt.Client) {
	broker := os.Getenv("MQTT_BROKER")
	portStr := os.Getenv("MQTT_PORT")
	mqtt_iid := os.Getenv("MQTT_IID")
	mqtt_key := os.Getenv("MQTT_KEY")
	mqtt_time_format := os.Getenv("MQTT_TIME_FORMAT")
	mqtt_value_roundstr := os.Getenv("MQTT_VALUE_ROUND")
	mqtt_value_round, err := strconv.Atoi(mqtt_value_roundstr)
	if err != nil {
		log.Fatalln(err.Error())
	}
	mqtt_qosstr := os.Getenv("MQTT_QOS")
	mqtt_qosint, err := strconv.Atoi(mqtt_qosstr)
	if err != nil {
		log.Fatalln(err.Error())
	}
	mqtt_qos := byte(mqtt_qosint)
	mqtt_retain_str := os.Getenv("MQTT_RETAIN")
	mqtt_retain, err := strconv.ParseBool(mqtt_retain_str)
	if err != nil {
		log.Fatalln(err.Error())
	}

	mqtt_heardbeat_intervalstr := os.Getenv("HEARTBEAT_INTERVAL")
	mqtt_heardbeat_interval, err := strconv.Atoi(mqtt_heardbeat_intervalstr)
	if err != nil {
		log.Fatalln(err.Error())
	}
	interval := int64(mqtt_heardbeat_interval)

	for {

		start_time := time.Now()

		cpuLoad, err := getCPULoad()
		if err != nil {
			log.Println("Error fetching CPU load:", err)
			continue
		}
		cpuLoad = round(cpuLoad, mqtt_value_round)

		// Get time
		serverTime := start_time

		message := Message{
			IID:   mqtt_iid,
			Key:   mqtt_key,
			Time:  serverTime.Format(mqtt_time_format),
			Value: cpuLoad,
		}

		// neo formatting to JSON
		jsonMessage, err := json.Marshal(message)
		if err != nil {
			log.Println("Error marshaling JSON:", err)
			continue
		}
		if !client.IsConnected() {
			log.Println("Error: MQTT client is not connected.")
			return
		}

		// Publish the mqqtt message
		token := client.Publish(topic+"/"+mqtt_iid+"/"+mqtt_key, mqtt_qos, mqtt_retain, jsonMessage)
		// token2 := client.Publish(topic,
		if token.Wait() && token.Error() != nil && !client.IsConnected() {
			// There was an error during publishing
			log.Println("Error publishing message:", token.Error())
		} else {
			// Message was delivered successfully
			fmt.Printf("Published CPU load: %f at %s to %s://%s:%s \n", cpuLoad, serverTime.Format("2006-01-02 15:04:05.999 -0700"), protocol, broker, portStr)
		}
		elapsed_time := time.Since(start_time).Milliseconds()
		remainingTime := interval - elapsed_time
		// fmt.Printf("Total Elapsed Time: %d milliseconds, remaining_time %d\n", elapsed_time, remainingTime)
		if remainingTime > 0 {
			time.Sleep(time.Duration(remainingTime) * time.Millisecond)
		}
	}
}

func getCPULoad() (float64, error) {
	percentages, err := cpu.Percent(time.Second, false) // true to get individual CPU percentages
	// gryyu, err := cpu.PercentWithContext(time.Second,)
	if err != nil {
		return 0, err
	}

	return percentages[0], nil
}

func NewTlsConfig() *tls.Config {
	certpool := x509.NewCertPool()
	ca, err := ioutil.ReadFile("ca_3.pem")
	if err != nil {
		log.Fatalln(err.Error())
	}
	certpool.AppendCertsFromPEM(ca)
	return &tls.Config{
		RootCAs: certpool,
	}
}

func round(f float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(f*pow) / pow
}
