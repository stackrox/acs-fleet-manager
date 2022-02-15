# Test Locally with the fleetshard Synchronizer

The communication between the fleetshard operator and the fleet-manager will be handled by the synchronizer module. The communication is established by the synchronizer module, so for it to work it needs to know the url of the control plane. However, when you run the fleet-manager locally and the fleetshard operator on a remote public OSD cluster, this will not work as there is no public URL for the synchronizer to use.

To solve this problem, it is recommended to use a service like [ngrok](https://ngrok.com/). Ngrok will be able to expose your local sever on the public internet.

## Steps

1. Register an account with [ngrok](https://ngrok.com/) first if you haven't done so. It is a free service.
2. Follow their [setup instructions](https://dashboard.ngrok.com/get-started/setup) to install and configure their CLI.
3. Open a terminal window and run `ngrok http 8000`. This will create a tunnel to the Ngrok service and you should see the forwarding urls to access your local server printed in the console. Copy the https one.
4. Before starting the fleet-manager, make sure you have added the right access token to the `observability-config-access.token` file in the local `secrets` directory (note: do not commit it!).
5. Start the fleet-manager with the following command:
   ```bash
   ./fleet-manager serve --public-host-url=<https forwarding url from Ngrok> <other flags>
   ```
6. You should also see access logs printed in the ngrok console, and the requests should be handled by the fleet-manager once it's started successfully.
