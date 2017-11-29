# koro: Container Routing toolkit

# What is 'koro'?

`koro` is a small tool which injects network routes into specified containers.
Target containers are docker container and linux ip netns namespace as well as
any network namespace given by pid.

# Build

`koro` is written in go, so following commands makes `koro` single binary.
Build and put it in your container host.

    git clone https://github.com/redhat-nfvpe/koro.git
    cd koro
    go get
    go build

# Syntax

    koro NS_SPEC address { add | del } ADDRESS dev STRING
    koro NS_SPEC route { add | del } ROUTE

    ROUTE := PREFIX NH
    NS_SPEC := { docker NAME | netns NAME | pid PID }
    NH := [ via ADDRESS ] [ dev STRING ]

# Example

    $ docker run -it --name koro_test1 <docker_images> <program> # launch container
    $ koro docker koro_test1 address add 127.0.0.3/24 dev lo # add ip address from container host

# Todo

- Document
- More options (mtu or some)
- Test, test, test!!!

# Acknowledgement
`koro` uses [go peg library](https://github.com/pointlander/peg) to parse CLI and
we thank the author, Andrew Snodgrass!

# Authors

- Tomofumi Hayashi (s1061123)
