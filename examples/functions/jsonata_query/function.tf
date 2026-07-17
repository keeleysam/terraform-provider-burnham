// JSONata query: aggregate and reshape a decoded structure in one expression.
output "order_summary" {
  value = provider::burnham::jsonata_query(
    {
      orders = [
        { product = "apple", qty = 3, price = 2 },
        { product = "pear", qty = 5, price = 4 },
      ]
    },
    "{ 'lines': $count(orders), 'total': $sum(orders.(qty * price)) }",
  )
  // → { lines = 2, total = 26 }
}
