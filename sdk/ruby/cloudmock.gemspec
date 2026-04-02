Gem::Specification.new do |s|
  s.name        = "cloudmock"
  s.version     = "1.0.0"
  s.summary     = "Local AWS emulation. 98 services."
  s.description = "CloudMock SDK — start a local AWS mock and get pre-configured clients."
  s.authors     = ["Viridian Inc"]
  s.homepage    = "https://github.com/Viridian-Inc/cloudmock"
  s.license     = "BSL-1.1"
  s.files       = ["lib/cloudmock.rb"]
  s.add_dependency "aws-sdk-core", ">= 3.0"
end
