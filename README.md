
Go to boot mode 

```
~/src/spidev-test/spidev_test -v -p "\x0C\x17"
```

Leave bootmode 
```
~/src/spidev-test/spidev_test -s 500000 -v -p "\x92\x04"
```
