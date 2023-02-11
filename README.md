[![codecov](https://codecov.io/github/fortio/log/branch/main/graph/badge.svg?token=LONYZDFQ7C)](https://codecov.io/github/fortio/log)

# Log

Fortio's log is simple logger built on top of go's default one with
additional opinionated levels similar to glog but simpler to use and configure.

It's been used for many years for Fortio's org Fortio project and more (under fortio.org/fortio/log package) but split out recently for standalone use, with the "flag polution" limited (as a library it doesn't include the flags, you configure it using apis).

```golang
log.Debugf() // Debug level
log.LogVf()  // Verbose level
log.Infof()  // Info/default level
log.Warnf()  // Warning level
log.Errf()   // Error level
log.Critf()  // Critical level (always logged even if level is set to max)
log.Fatalf() // Fatal level - program will panic/exit
```

See the `Config` object for options like whether to include line number and file name of caller or not etc
