```
for f in $(find -not -path '*/[@.]*' -type f); do curl -v http://localhost:8080/$f -T $f; done
```
