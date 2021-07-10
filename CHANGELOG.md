# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased](../../releases/tag/X.Y.Z)

## [1.4.3](../../releases/tag/1.4.3) - 2021-07-10

### Fixed

- Unhandled type error when data field exists and data value is null ([#10](../../issues/10))

## [1.4.2](../../releases/tag/1.4.2) - 2021-06-12

### Fixed

- panic: runtime error: invalid memory address or nil pointer dereference bug ([#65](../../issues/65))
- CLI option -port not working bug ([#84](../../issues/84))
- Bump github.com/gogo/protobuf from 1.3.1 to 1.3.2
- Bump gopkg.in/yaml.v2 from 2.3.0 to 2.4.0
- Bump golang from 1.15.5 to 1.16.5

## [1.4.1](../../releases/tag/1.4.1) - 2020-08-29

### Fixed

- Panic when there is a query execution related error (#54)
- Minor refactoring