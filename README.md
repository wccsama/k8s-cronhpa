# k8s-cronhpa

`定时伸缩deploy sts，兼容HPA`

`TODO:`
```
add makefile
add yaml
fix request in controller
```

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
```
`build`
`deploy CRD`
`deploy controller`
