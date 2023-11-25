# Go Heard_beat


## cross platform buliding (from windows to linux)
### 1. open powershell and run below cmd :
### 2. $Env:GOOS = "linux"; $Env:GOARCH = "amd64"
### 3. go build -o daq_heard_bead .\main.go

##if you are build from windows machine(linux to linux machine)
### 1. open shell in linux
### 2. go build -o daq_heard_bead .\main.go


##if you are build from linux machine(windows to windows machine)
### 1. open cmd in windows
### 2. go build -o daq_heard_bead .\main.go


###websocked still pending

