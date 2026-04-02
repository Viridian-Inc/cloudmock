require "minitest/autorun"
require_relative "../lib/cloudmock"

class CloudMockTest < Minitest::Test
  def test_server_attributes
    server = CloudMock::Server.new(port: 9999)
    assert_equal 9999, server.port
    assert_equal "http://localhost:9999", server.endpoint
  end

  def test_auto_port
    server = CloudMock::Server.new
    assert server.port > 0
    assert_match %r{^http://localhost:\d+$}, server.endpoint
  end
end
