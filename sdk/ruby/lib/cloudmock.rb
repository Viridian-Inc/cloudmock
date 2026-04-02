# frozen_string_literal: true

require "socket"
require "net/http"
require "uri"

module CloudMock
  class Server
    attr_reader :port, :endpoint

    def initialize(port: nil, region: "us-east-1", profile: "minimal")
      @port = port || find_free_port
      @region = region
      @profile = profile
      @endpoint = "http://localhost:#{@port}"
      @pid = nil
    end

    def start
      return self if @pid

      @pid = spawn(
        { "CLOUDMOCK_PROFILE" => @profile, "CLOUDMOCK_IAM_MODE" => "none" },
        "cloudmock", "--port", @port.to_s,
        out: File::NULL, err: File::NULL
      )

      at_exit { stop }
      wait_for_ready(30)
      self
    end

    def stop
      return unless @pid
      Process.kill("TERM", @pid) rescue nil
      Process.wait(@pid) rescue nil
      @pid = nil
    end

    def aws_config
      {
        endpoint: @endpoint,
        region: @region,
        credentials: Aws::Credentials.new("test", "test"),
        force_path_style: true
      }
    end

    private

    def find_free_port
      server = TCPServer.new("127.0.0.1", 0)
      port = server.addr[1]
      server.close
      port
    end

    def wait_for_ready(timeout)
      deadline = Time.now + timeout
      loop do
        raise "CloudMock did not start in time" if Time.now > deadline
        begin
          TCPSocket.new("127.0.0.1", @port).close
          return
        rescue Errno::ECONNREFUSED
          sleep 0.1
        end
      end
    end
  end

  def self.start(**opts)
    Server.new(**opts).start
  end
end
