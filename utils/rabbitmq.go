package utils

/*
define a light rabbit framwork whit featrues like:
1.exchang,channal,messages are all seted to be presistent
2.one connection with one channal
3.channel.Qos(1, 0, false)
4.reconnect when socket close.
*/
import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/streadway/amqp"
)

type Pdcser struct {
	uri,
	exchangeName,
	queueName,
	rkey string
	conn  *amqp.Connection
	chanl *amqp.Channel
}

func Newpdcser(amqpuri, exname, qNname, rk string) (*Pdcser, error) {
	pc := &Pdcser{
		uri:          amqpuri,
		exchangeName: exname,
		queueName:    qNname,
		rkey:         rk,
	}
	err := pc.init()
	if err != nil {
		return nil, err
	}
	return pc, nil
}
func (p *Pdcser) init() error {
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
	if err != nil {
		return err
	}

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
	if err != nil {
		return err
	}
	err = channel.Qos(1, 0, false) //确保公平分发
	if err != nil {
		return err
	}
	//创建一个queue
	q, err := channel.QueueDeclare(
		p.queueName, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return err
	}
	//绑定
	err = channel.QueueBind(q.Name, p.rkey, p.exchangeName, false, nil)
	if err != nil {
		return err
	}
	return nil
}
func (p *Pdcser) Publish(msg []byte) error {
	var err error
	select {
	case <-p.conn.NotifyClose(make(chan *amqp.Error)):
		log.Println("connection to MQServer closed,reconnecting...")
		if err = p.init(); err != nil {
			return err
		}
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
func (p *Pdcser) Comsumer(ch chan []byte) error {
recon:
	msgs, err := p.chanl.Consume(
		p.queueName, // queue
		"",          // consumer
		false,       // auto-ack,不是真正的comsumer确认,需要应用层主动回复ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}
	ech := make(chan *amqp.Error)
	for {
		select {
		case <-p.conn.NotifyClose(ech):
			log.Println("connection to MQServer closed,reconnecting...")
			if err := p.init(); err != nil {
				return err
			}
			goto recon
		case d := <-msgs:
			ch <- d.Body
			d.Ack(true)
		}
	}

}
func (p *Pdcser) Close() {
	_ = p.chanl.Close()
	_ = p.conn.Close()
}
