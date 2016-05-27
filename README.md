# ifacemaker

This is a development helper that generates an interface from a structure methods.
My primary use case is to generate go-mocks from interfaces. So, unit testing gets easier.


```
~ » ifacemaker --help
Options:

  -h, --help              display help information
  -f, --file             *go source file with the structure
  -s, --struct           *Structure type name to look for
  -i, --iface            *Exported interface name
  -p, --pkg              *Package name
  -d, --nodoc[=false]     Copy docs from methods
  -o, --output            Output file name. If not provided, result will be printed to stdout
```
