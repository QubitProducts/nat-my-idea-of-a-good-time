.PHONY: deploy-production deploy-staging

deploy-production:
	baton -d -e production

deploy-staging:
	baton -d -e staging
