package config

import (
	"errors"
	"strings"

	"github.com/oasislabs/developer-gateway/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type MailboxProvider string

const (
	MailboxRedisSingle  MailboxProvider = "redis-single"
	MailboxRedisCluster MailboxProvider = "redis-cluster"
	MailboxMem          MailboxProvider = "mem"
)

type MailboxConfig struct {
	Binder
	Provider string
	Mailbox  Mailbox
}

func (c *MailboxConfig) Log(fields log.Fields) {
	fields.Add("mailbox.provider", c.Provider)

	if c.Mailbox != nil {
		c.Mailbox.Log(fields)
	}
}

func (c *MailboxConfig) Configure(v *viper.Viper) error {
	c.Provider = v.GetString("mailbox.provider")
	if len(c.Provider) == 0 {
		return errors.New("mailbox.provider must be set. " +
			"Options are " + string(MailboxMem) +
			", " + string(MailboxRedisSingle) +
			", " + string(MailboxRedisCluster) + ".")
	}

	switch MailboxProvider(c.Provider) {
	case MailboxMem:
		c.Mailbox = &MailboxMemConfig{}
		return c.Mailbox.(*MailboxMemConfig).Configure(v)
	case MailboxRedisSingle:
		c.Mailbox = &MailboxRedisSingleConfig{}
		return c.Mailbox.(*MailboxRedisSingleConfig).Configure(v)
	case MailboxRedisCluster:
		c.Mailbox = &MailboxRedisClusterConfig{}
		return c.Mailbox.(*MailboxRedisClusterConfig).Configure(v)
	default:
		return errors.New("unknown mailbox.provider set. " +
			"Options are " + string(MailboxMem) +
			", " + string(MailboxRedisSingle) +
			", " + string(MailboxRedisCluster) + ".")
	}
}

func (c *MailboxConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("mailbox.provider", "mem",
		"provider for the mailbox service. "+
			"Options are "+string(MailboxMem)+
			", "+string(MailboxRedisSingle)+
			", "+string(MailboxRedisCluster)+".")

	if err := (&MailboxRedisSingleConfig{}).Bind(v, cmd); err != nil {
		return err
	}
	if err := (&MailboxRedisClusterConfig{}).Bind(v, cmd); err != nil {
		return err
	}
	if err := (&MailboxMemConfig{}).Bind(v, cmd); err != nil {
		return err
	}

	return nil
}

type Mailbox interface {
	log.Loggable
	Binder
	ID() MailboxProvider
}

type MailboxRedisSingleConfig struct {
	Addr string
}

func (c *MailboxRedisSingleConfig) Log(fields log.Fields) {
	fields.Add("mailbox.redis_single.addr", c.Addr)
}

func (c *MailboxRedisSingleConfig) ID() MailboxProvider {
	return MailboxRedisSingle
}

func (c *MailboxRedisSingleConfig) Configure(v *viper.Viper) error {
	c.Addr = v.GetString("mailbox.redis_single.addr")
	if len(c.Addr) == 0 {
		return errors.New("mailbox.redis_single.addr must be set")
	}

	return nil
}

func (c *MailboxRedisSingleConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().String("mailbox.redis_single.addr", "127.0.0.1:6379", "redis instance address")
	return nil
}

type MailboxRedisClusterConfig struct {
	Addrs []string
}

func (c *MailboxRedisClusterConfig) Log(fields log.Fields) {
	fields.Add("mailbox.redis_cluster.addrs", strings.Join(c.Addrs, ","))
}

func (c *MailboxRedisClusterConfig) ID() MailboxProvider {
	return MailboxRedisCluster
}

func (c *MailboxRedisClusterConfig) Configure(v *viper.Viper) error {
	c.Addrs = v.GetStringSlice("mailbox.redis_cluster.addrs")
	if len(c.Addrs) == 0 {
		return errors.New("mailbox.redis_cluster.addrs must be set")
	}

	return nil
}

func (c *MailboxRedisClusterConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	cmd.PersistentFlags().StringArray(
		"mailbox.redis_cluster.addrs",
		[]string{"127.0.0.1:6379"},
		"array of addresses for bootstrap redis instances in the cluster")
	return nil
}

type MailboxMemConfig struct{}

func (c *MailboxMemConfig) Log(fields log.Fields) {}

func (c *MailboxMemConfig) ID() MailboxProvider {
	return MailboxMem
}

func (c *MailboxMemConfig) Configure(v *viper.Viper) error {
	return nil
}

func (c *MailboxMemConfig) Bind(v *viper.Viper, cmd *cobra.Command) error {
	return nil
}
