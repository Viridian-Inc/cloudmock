//! CloudMock — Local AWS emulation for Rust tests.
//!
//! # Example
//! ```no_run
//! use cloudmock::CloudMock;
//!
//! #[tokio::test]
//! async fn test_s3() {
//!     let cm = CloudMock::start().await.unwrap();
//!     let config = aws_config::defaults(aws_config::BehaviorVersion::latest())
//!         .endpoint_url(cm.endpoint())
//!         .credentials_provider(aws_credential_types::Credentials::new("test", "test", None, None, "cloudmock"))
//!         .region(aws_config::Region::new("us-east-1"))
//!         .load()
//!         .await;
//!     let s3 = aws_sdk_s3::Client::new(&config);
//!     // Use s3 client...
//!     cm.stop().await;
//! }
//! ```

use std::net::TcpListener;
use std::process::Stdio;
use tokio::process::{Child, Command};
use tokio::time::{sleep, Duration, Instant};

pub struct CloudMock {
    process: Child,
    port: u16,
}

pub struct CloudMockOptions {
    pub port: Option<u16>,
    pub region: String,
    pub profile: String,
}

impl Default for CloudMockOptions {
    fn default() -> Self {
        Self {
            port: None,
            region: "us-east-1".to_string(),
            profile: "minimal".to_string(),
        }
    }
}

impl CloudMock {
    /// Start a CloudMock instance with default options.
    pub async fn start() -> Result<Self, Box<dyn std::error::Error>> {
        Self::start_with(CloudMockOptions::default()).await
    }

    /// Start a CloudMock instance with custom options.
    pub async fn start_with(opts: CloudMockOptions) -> Result<Self, Box<dyn std::error::Error>> {
        let port = opts.port.unwrap_or_else(free_port);

        let process = Command::new("cloudmock")
            .args(["--port", &port.to_string()])
            .env("CLOUDMOCK_PROFILE", &opts.profile)
            .env("CLOUDMOCK_IAM_MODE", "none")
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .spawn()?;

        let cm = Self { process, port };
        cm.wait_ready(Duration::from_secs(30)).await?;
        Ok(cm)
    }

    /// The endpoint URL for this instance.
    pub fn endpoint(&self) -> String {
        format!("http://localhost:{}", self.port)
    }

    /// The port this instance is listening on.
    pub fn port(&self) -> u16 {
        self.port
    }

    /// Stop the CloudMock instance.
    pub async fn stop(mut self) {
        let _ = self.process.kill().await;
    }

    async fn wait_ready(&self, timeout: Duration) -> Result<(), Box<dyn std::error::Error>> {
        let deadline = Instant::now() + timeout;
        loop {
            if Instant::now() > deadline {
                return Err("CloudMock did not start in time".into());
            }
            match reqwest_lite_check(self.port).await {
                Ok(()) => return Ok(()),
                Err(_) => sleep(Duration::from_millis(100)).await,
            }
        }
    }
}

impl Drop for CloudMock {
    fn drop(&mut self) {
        // Best-effort kill on drop
        let _ = self.process.start_kill();
    }
}

fn free_port() -> u16 {
    TcpListener::bind("127.0.0.1:0")
        .expect("bind to free port")
        .local_addr()
        .expect("local addr")
        .port()
}

async fn reqwest_lite_check(port: u16) -> Result<(), Box<dyn std::error::Error>> {
    let stream = tokio::net::TcpStream::connect(format!("127.0.0.1:{}", port)).await?;
    drop(stream);
    Ok(())
}
