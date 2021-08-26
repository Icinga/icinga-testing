# Icinga Testing

![Icinga Logo](https://icinga.com/wp-content/uploads/2014/06/icinga_logo.png)

## About

This repository contains helpers to facilitate performing integration tests between components of the Icinga stack using
the [Go `testing` package](https://pkg.go.dev/testing). The general idea is to write test cases in Go that can
dynamically spawn individual components as required, connect them and then perform checks on this setup. This is
currently implemented by using the Docker API to start and stop containers locally as required by the tests.

## License

The contents of this repository are licensed under the terms of the GNU General Public License Version 2, you will find
a copy of this license in the LICENSE file included in it.
