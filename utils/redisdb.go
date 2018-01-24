package utils

import (
	"errors"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
)

type GetSFnumsFromRedisResp struct {
	ErrStatus  string
	Succ, Fail int
}
type GetResultFromRedisResp struct {
	ErrStatus  string
	Succ, Fail map[string]string
}
type GetAgentResultFromRedisResp struct {
	ErrStatus string
	Res       string
}

//redis连接池机制,当并发量大于最大rmaxactive时,可能会出现"连接池"满的报错,只能等待其它并发进程释放连接资源后续的请求才能继续;另并发量较大时会导致redis服务端time_wait较多,OS层面可配置连接的快速回收
func Newredisclient(Redisconstr, Redispass string, RedisDB, RMaxIdle, RMaxActive int) (*redis.Pool, error) {
	RedisClient := &redis.Pool{ //以连接池的方式进行redis操作
		Dial: func() (redis.Conn, error) {
			//c, err := redis.Dial("tcp", Redisconstr, redis.DialPassword(Redispass))
			c, err := redis.Dial("tcp", Redisconstr)
			if err != nil {
				return nil, err
			}
			c.Do("SELECT", RedisDB)
			return c, nil
		},
		MaxIdle:     int(RMaxIdle),
		MaxActive:   int(RMaxActive),
		IdleTimeout: time.Second * 3600,
	}
	rc := RedisClient.Get()
	defer rc.Close()
	rp, e := redis.String(rc.Do("ping"))
	if e != nil || rp != "PONG" {
		return nil, errors.New(e.Error() + rp)
	}
	return RedisClient, nil
}
func GetSFnumsFromRedis(uuid string, rc redis.Conn) (rs GetSFnumsFromRedisResp) { //通过uuid获取成功失败数量,用于前端刷新
	defer rc.Close()
	var err error
	rs.Succ, err = redis.Int(rc.Do("hlen", uuid+":true"))
	if err != nil {
		rs.ErrStatus = err.Error()
	}
	rs.Fail, err = redis.Int(rc.Do("hlen", uuid+":false"))
	if err != nil {
		rs.ErrStatus = err.Error()
	}
	return rs
}
func GetResultFromRedis(uuid string, rc redis.Conn) (rs GetResultFromRedisResp) { //通过uuid拉取所有agent结果信息,用于前端刷新
	defer rc.Close()
	var err error
	rs.Succ, err = redis.StringMap(rc.Do("hgetall", uuid+":true"))
	if err != nil {
		rs.ErrStatus = err.Error()
	}
	rs.Fail, err = redis.StringMap(rc.Do("hgetall", uuid+":false"))
	if err != nil {
		rs.ErrStatus = err.Error()
	}
	return rs
}
func GetAgentResultFromRedis(uuid, ip string, rc redis.Conn) (rs GetAgentResultFromRedisResp) { //通过uuidh和ip拉取某个agent结果信息
	defer rc.Close()
	var err error
	rs.Res, err = redis.String(rc.Do("hget", uuid+":true", ip))
	if err == nil {
		return rs
	}
	rs.Res, err = redis.String(rc.Do("hget", uuid+":false", ip))
	if err != nil {
		rs.ErrStatus = err.Error()
	}
	return rs
}
func GetAgentResultInSucc(uuid, ip string, rc redis.Conn) (rs GetAgentResultFromRedisResp) {
	defer rc.Close()
	var err error
	rs.Res, err = redis.String(rc.Do("hget", uuid+":true", ip))
	if err != nil {
		rs.ErrStatus = err.Error()
	}
	return rs
}
func GetAgentResultInFail(uuid, ip string, rc redis.Conn) (rs GetAgentResultFromRedisResp) {
	defer rc.Close()
	var err error
	rs.Res, err = redis.String(rc.Do("hget", uuid+":false", ip))
	if err != nil {
		rs.ErrStatus = err.Error()
	}
	return rs
}
func DelResponseFromRedis(uuid string, rc redis.Conn) error {
	defer rc.Close()
	if _, e := redis.Int(rc.Do("del", uuid+":true")); e != nil && e != redis.ErrNil {
		return e
	}
	if _, e := redis.Int(rc.Do("del", uuid+":false")); e != nil && e != redis.ErrNil {
		return e
	}
	return nil
}
func Writeresponserun(msg *RcsTaskResp, rc redis.Conn) error {
	defer rc.Close()
	if _, e := redis.Int(rc.Do("hset", msg.Runid+":"+strconv.FormatBool(msg.Flag), msg.AgentIP, msg.Result)); e != nil {
		return e
	}
	return nil
}

//////////////////////////////////////////
func WriteAgentinfo(jsvip string, msg *AgentSyncMsg, rc redis.Conn) error {
	defer rc.Close()

	switch msg.Op {
	case "add":
		if _, e := redis.Int(rc.Do("sadd", jsvip, msg.Agentip)); e != nil {
			return e
		}
	case "del":
		if _, e := redis.Int(rc.Do("srem", jsvip, msg.Agentip)); e != nil {
			return e
		}
	default:
	}
	return nil
}
