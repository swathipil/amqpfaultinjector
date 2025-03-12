using System.Linq;
using Azure.Messaging.ServiceBus;
using Azure.Identity;
using System.Net;
using System.Security.Cryptography.X509Certificates;
using System.Net.Security;
using Azure.Messaging.EventHubs.Producer;

var pairs = File.ReadAllLines(".env")
    .Select(line => line.Split("=", 2))
    .Where(parts => parts.Length == 2);

ServicePointManager.ServerCertificateValidationCallback = (object sender, X509Certificate? certificate, X509Chain? chain, SslPolicyErrors errors) =>
{
    return true;
};

var endpoint = pairs.First(pair => pair[0] == "EVENTHUBS_ENDPOINT")[1];
var entity = pairs.First(pair => pair[0] == "EVENTHUBS_HUBNAME")[1];
var tokenCred = new DefaultAzureCredential();

await using var producer = new EventHubProducerClient(endpoint, entity, tokenCred, new EventHubProducerClientOptions()
{
    ConnectionOptions = new Azure.Messaging.EventHubs.EventHubConnectionOptions()
    {
        CertificateValidationCallback = (object sender, X509Certificate? certificate, X509Chain? chain, SslPolicyErrors sslPolicyErrors) =>
        {
            return true;
        },
        CustomEndpointAddress = new Uri("sb://localhost:5671"),
    }
});

Console.WriteLine($"Connecting to {endpoint}/{entity}");
var batch = await producer.CreateBatchAsync();
_ = batch.TryAdd(new Azure.Messaging.EventHubs.EventData("hello world"));

Console.WriteLine("Sending message");
await producer.SendAsync(batch);

Console.WriteLine("Done");
