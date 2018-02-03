package utils

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/streadway/amqp"
)

type pdcser struct {
	uri,
	exchangeName,
	queueName,
	rkey string
	conn  *amqp.Connection
	chanl *amqp.Channel
}

func Newpdcser(amqpuri, exname, qNname, rk string) *pdcser {
	pc := &pdcser{
		uri:          amqpuri,
		exchangeName: exname,
		queueName:    qNname,
		rkey:         rk,
	}
	pc.init()
	return pc

}
func (p *pdcser) init() {
	var connection *amqp.Connection
	var err error
	//建立一个连接
	for {
		connection, err = amqp.Dial(p.uri)
		if err != nil {
			log.Println("amqp.Dial error,reconnect..:", err)
			time.Sleep(time.Second * time.Duration(1+rand.New(rand.NewSource(time.Now().UnixNano())).Intn(10)))
			continue
		}
		break
	}
	p.conn = connection
	//defer connection.Close()
	//创建一个Channel
	channel, err := connection.Channel()
	failOnError(err, "Failed to open a channel")
	//defer channel.Close()
	p.chanl = channel
	//创建一个exchange
	err = channel.ExchangeDeclare(
		p.exchangeName,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare an exchange")
	err = channel.Qos(1, 0, false) //确保公平分发
	failOnError(err, "Failed to set qos")
	//创建一个queue
	q, err := channel.QueueDeclare(
		p.queueName, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	failOnError(err, "Failed to declare a queue")
	//绑定
	err = channel.QueueBind(q.Name, p.rkey, p.exchangeName, false, nil)
	failOnError(err, "Failed to bind exchange and queue")
}
func (p *pdcser) Publish(msg []byte) error {
	var err error
	select {
	case <-p.conn.NotifyClose(make(chan *amqp.Error)):
		log.Println("connection to MQServer closed,reconnecting...")
		p.init()
		err = errors.New("Reconnected")
	default:
		err = p.chanl.Publish(
			p.exchangeName, // exchange
			p.rkey,         // routing key
			false,          // mandatory
			false,          // immediate
			amqp.Publishing{
				Headers:         amqp.Table{},
				ContentType:     "text/plain",
				ContentEncoding: "",
				DeliveryMode:    2, //msg Persistent
				Body:            msg,
			})
	}
	return err
}
func (p *pdcser) Comsumer(msgs <-chan amqp.Delivery) error {
	var err error
	//msgch := (<-chan amqp.Delivery)(msgs)
	msgs, err = p.chanl.Consume(
		p.queueName, // queue
		"",          // consumer
		false,       // auto-ack,不是真正的comsumer确认,需要应用层主动回复ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)

	return err
}
func (p *pdcser) Close() {
	_ = p.chanl.Close()
	_ = p.conn.Close()
}
func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
