Parses a Java [`.properties`](https://en.wikipedia.org/wiki/.properties) file body into an object. Comments (`#` and `!`), `=`/`:`/whitespace separators, line continuation via trailing `\`, and `\uXXXX` Unicode escapes are all handled per the standard `java.util.Properties` semantics.

By default property expansion (`${other.key}` substitution) is disabled, so values are returned exactly as written. All values are returned as strings.

Backed by [magiconair/properties](https://github.com/magiconair/properties), an actively-maintained Go implementation.

**Common uses:** ingesting Spring/Quarkus `application.properties`, JBoss/WildFly server config, or any JVM-shop runtime configuration.