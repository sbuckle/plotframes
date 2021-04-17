require 'rake'
require 'rake/clean'

TARGET = Rake::FileList.new("plotframes*")

task default: %w[build]

desc 'Build the executable'
task :build => :clean do
  sh "go build"
end

desc 'Install the executable'
task :install do
  sh "go install"
end

CLEAN << TARGET
