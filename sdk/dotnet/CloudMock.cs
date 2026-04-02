using System;
using System.Diagnostics;
using System.Net.Sockets;
using System.Net;
using System.Threading;

namespace CloudMock
{
    public class CloudMockServer : IDisposable
    {
        private Process _process;
        public int Port { get; }
        public string Endpoint => $"http://localhost:{Port}";
        public string Region { get; }

        public CloudMockServer(int port = 0, string region = "us-east-1", string profile = "minimal")
        {
            Port = port > 0 ? port : FindFreePort();
            Region = region;

            _process = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = "cloudmock",
                    Arguments = $"--port {Port}",
                    RedirectStandardOutput = true,
                    RedirectStandardError = true,
                    UseShellExecute = false,
                    CreateNoWindow = true,
                    Environment = {
                        ["CLOUDMOCK_PROFILE"] = profile,
                        ["CLOUDMOCK_IAM_MODE"] = "none"
                    }
                }
            };
            _process.Start();
            WaitForReady(TimeSpan.FromSeconds(30));
        }

        public void Dispose()
        {
            if (_process != null && !_process.HasExited)
            {
                _process.Kill();
                _process.WaitForExit(5000);
            }
            _process?.Dispose();
        }

        private void WaitForReady(TimeSpan timeout)
        {
            var deadline = DateTime.UtcNow + timeout;
            while (DateTime.UtcNow < deadline)
            {
                try
                {
                    using var client = new TcpClient();
                    client.Connect("127.0.0.1", Port);
                    return;
                }
                catch (SocketException)
                {
                    Thread.Sleep(100);
                }
            }
            throw new TimeoutException("CloudMock did not start in time");
        }

        private static int FindFreePort()
        {
            var listener = new TcpListener(IPAddress.Loopback, 0);
            listener.Start();
            int port = ((IPEndPoint)listener.LocalEndpoint).Port;
            listener.Stop();
            return port;
        }
    }
}
