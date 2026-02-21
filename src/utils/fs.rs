pub fn sanitize_name(name: &str) -> String {
    name.replace(['/', '\\', ' ', ':'], "_")
}
