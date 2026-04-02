using Xunit;

namespace CloudMock.Tests
{
    public class CloudMockTests
    {
        [Fact]
        public void Endpoint_Format()
        {
            var uri = new System.Uri("http://localhost:4566");
            Assert.Equal("localhost", uri.Host);
            Assert.Equal(4566, uri.Port);
        }
    }
}
