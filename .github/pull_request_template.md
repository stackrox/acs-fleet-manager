## Description
<!-- Please include a summary of the change and a link to the JIRA ticket. Please add any additional motivation and context as needed. Screenshots are also welcome -->

## Checklist (Definition of Done)
<!-- Please strikethrough options not relevant using two tildes ~~Text~~. Do not delete non relevant options -->
- [ ] Unit and integration tests added
- [ ] Added test description under `Test manual`
- [ ] Documentation added if necessary (i.e. changes to dev setup, test execution, ...)
- [ ] CI and all relevant tests are passing
- [ ] Add the ticket number to the PR title if available, i.e. `ROX-12345: ...`
- [ ] Discussed security and business related topics privately. Will move any security and business related topics that arise to private communication channel.

## Test manual

**TODO:** Add manual testing efforts

```
# To run tests locally run:
make db/teardown db/setup db/migrate
make ocm/setup OCM_OFFLINE_TOKEN=<ocm-offline-token> OCM_ENV=development
make verify lint binary test test/integration
```
