# k8s-cronhpa
`定时伸缩deploy sts，兼容HPA` 
`CronHPA is compatible with HPA and scales workloads (e.g. deployment, statefulset) periodically`

# example
`CronHPA` example:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cronhpa-wcc-test
spec:
  replicas: 1
  selector:
    matchLabels:
      name: cronhpa-wcc-test
  template:
    metadata:
      name: cronhpa-wcc-test
      labels:
        name: cronhpa-wcc-test
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
          - containerPort: 80
---
apiVersion: wcc.io/v1
kind: CronHPA
metadata:
  name: cronhpa-sample
  namespace: default
spec:
   scaleTargetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: cronhpa-wcc-test
   excludeDates:
     - "*/2 * * * *"
   crons:
   - name: "scale-down"
     schedule: "*/2 * * * *"
     targetSize: 1
     runOnce: true
   - name: "scale-up"
     schedule: "*/3 * * * *"
     targetSize: 4
     runOnce: false
     # cron https://en.wikipedia.org/wiki/Cron
```

# build
```
set HUB
make build
make docker-build
```

# deploy controller && example
```
kubectl apply -f crd.yaml
kubectl apply -f cronhpa.yaml
kubectl apply -f example.yaml
```

