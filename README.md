In `bash`, the `-x` flag (or equivalently `set -x`) activates debugging mode. This is useful to trace the execution of a
script, but it does so for every command run, including files `source`-d from the script in which `set -x` is set. This
often leads to incredibly verbose and hard to follow output.

Instead, in the spirit of the output seen in CI systems like Travis, we use `vbash`:

```
> cat <<EOD > test.sh
#!/path/to/vbash

echo "Hello world"
EOD
> chmod +x test.sh
> ./test.sh
```

gives the output:

```
$ echo "Hello world"
Hello world
```
