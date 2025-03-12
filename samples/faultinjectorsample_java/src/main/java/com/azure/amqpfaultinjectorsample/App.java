package com.azure.amqpfaultinjectorsample;

import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.util.Arrays;

import org.apache.qpid.proton.engine.SslDomain;

import com.azure.messaging.eventhubs.EventData;
import com.azure.messaging.eventhubs.EventHubClientBuilder;

/**
 * Hello world!
 */
public final class App {
    private App() {
    }

    /**
     * Says hello to the world.
     * 
     * @param args The arguments of the program.
     * @throws SecurityException
     * @throws NoSuchMethodException
     * @throws InvocationTargetException
     * @throws IllegalArgumentException
     * @throws IllegalAccessException
     */
    public static void main(String[] args) throws NoSuchMethodException, SecurityException, IllegalAccessException,
            IllegalArgumentException, InvocationTargetException {
        var tokenCred = new com.azure.identity.DefaultAzureCredentialBuilder().build();

        var builder = new EventHubClientBuilder()
                .credential("<event hub namespace>", "eventhub", tokenCred)
                .customEndpointAddress("amqp://127.0.0.1");

        // TODO: I know this makes me a bad person. Temporary until I add in certificate
        // imports.
        Method m = builder.getClass().getDeclaredMethod("verifyMode", SslDomain.VerifyMode.class);
        m.setAccessible(true);
        builder = (EventHubClientBuilder) m.invoke(builder, SslDomain.VerifyMode.ANONYMOUS_PEER); // Now it's OK

        try (var producerClient = builder.buildProducerClient()) {
            producerClient.send(Arrays.asList(new EventData("hello world")));
        }
    }
}
