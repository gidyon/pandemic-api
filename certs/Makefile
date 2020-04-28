CA_LOCATION := /etc/kubernetes/pki

ifdef version
	VERSION=-$(version)
else
	VERSION=
endif

create_messaging_tls:
	kubectl create secret generic messaging-tls$(VERSION) --from-file=cert=messaging/cert.pem --from-file=key=messaging/key.pem
create_location_tls:
	kubectl create secret generic location-tls$(VERSION) --from-file=cert=location/cert.pem --from-file=key=location/key.pem
create_tracing_tls:
	kubectl create secret generic tracing-tls$(VERSION) --from-file=cert=tracing/cert.pem --from-file=key=tracing/key.pem
create_gateway_tls:
	sudo kubectl create secret generic gateway-tls$(VERSION) --from-file=cert=gateway/gateway.crt --from-file=key=gateway/gateway.key
create_restful_tls:
	kubectl create secret generic restful-tls$(VERSION) --from-file=cert=restful/cert.pem --from-file=key=restful/key.pem

create_all: create_messaging_tls create_location_tls create_tracing_tls create_gateway_tls

delete_tls_secrets: 
	kubectl delete secret  messaging-tls$(VERSION) location-tls$(VERSION) tracing-tls$(VERSION) gateway-tls$(VERSION)