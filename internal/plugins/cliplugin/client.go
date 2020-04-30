package cliplugin

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	hclog "github.com/hashicorp/go-hclog"
	plugin "github.com/hashicorp/go-plugin"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	proto "github.com/newrelic/newrelic-cli/internal/plugins/protoDef"
)

const (
	pluginType       = "cli_plugin"
	magicCookieKey   = "NEWRELIC_CLI_PLUGIN"
	magicCookieValue = "4951e1a8-27fa-4fc0-b04c-308fc3ed5799"
)

var (
	handshakeConfig = plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   magicCookieKey,
		MagicCookieValue: magicCookieValue,
	}
	pluginMap = map[string]plugin.Plugin{
		pluginType: &CLIPlugin{},
	}
	errCh = make(chan error)
)

// Client is used for communicating with a CLI plugin.
type Client struct {
	pb         proto.CLIClient
	pluginHost *plugin.Client
}

// ClientOptions represents the options to be passed to the client.
type ClientOptions struct {
	LogLevel string
	Command  string
	Args     []string
}

// NewClient creates a new client for communicating with a CLI plugin.
func NewClient(opts *ClientOptions) *Client {
	if opts.LogLevel == "" {
		opts.LogLevel = "Info"
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Level: hclog.LevelFromString(opts.LogLevel),
	})

	pluginHost := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  handshakeConfig,
		Plugins:          pluginMap,
		Cmd:              exec.Command(opts.Command, opts.Args...),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           logger,
	})

	rpcClient, err := pluginHost.Client()
	if err != nil {
		log.Fatalf(err.Error())
	}

	raw, err := rpcClient.Dispense(pluginType)
	if err != nil {
		log.Fatalf(err.Error())
	}

	client := raw.(*Client)
	client.pluginHost = pluginHost

	return client
}

// Kill ends the plugin host process and cleans up remaining resources.
func (c *Client) Kill() {
	fmt.Println("KILL...")

	c.pluginHost.Kill()
}

// Exec allows for executing a given subcommand.
func (c *Client) Exec(command string, args []string) error {
	stream, err := c.pb.Exec(context.Background())

	if err != nil {
		return err
	}

	err = stream.Send(&proto.ExecRequest{
		Command: command,
		Args:    args,
	})

	if err != nil {
		return err
	}

	var stdout, stderr bytes.Buffer

	go handleStdin(stream)

	for {
		chunk, err := stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			break
		}

		stdout.Write(chunk.Stdout)
		_, err = io.Copy(os.Stdout, &stdout)
		if err != nil {
			log.Fatal(err)
		}

		stderr.Write(chunk.Stderr)
		_, err = io.Copy(os.Stderr, &stderr)
		if err != nil {
			log.Fatal(err)
		}
	}

	close(errCh)

	return nil
}

// Exec allows for executing a given subcommand.
func (c *Client) ExecSimple(command string, args []string) (string, error) {
	execReqSimple := proto.ExecRequestSimple{
		Command: command,
		Args:    args,
	}

	fmt.Println("ExecSimple...................................")

	resp, err := c.pb.ExecSimple(context.Background(), &execReqSimple)

	fmt.Printf("ExecSimple Resp: %+v \n", resp)
	fmt.Printf("ExecSimple Error: %+v \n", err)

	if err != nil {
		return "", err
	}

	return resp.Output, nil
}

func handleStdin(stream proto.CLI_ExecClient) {
	reader := bufio.NewReader(os.Stdin)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			errCh <- err
		}

		err = stream.Send(&proto.ExecRequest{
			Stdin: []byte{b},
		})

		if err != nil {
			errCh <- err
		}
	}
}

// CLIPlugin represents a gRPC-aware plugin powered by go-plugin.
// It satisfies the plugin.GRPCPlugin interface.
type CLIPlugin struct {
	plugin.Plugin
}

// GRPCServer creates a gRPC server for running a plugin.
// This is currently not implemented, but is here to satisfy the underlying interface.
func (p *CLIPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

// GRPCClient creates a gRPC client for communicating with a plugin.
func (p *CLIPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &Client{pb: proto.NewCLIClient(c)}, nil
}
